package main

import (
	"database/sql"
	"errors"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const fileDir = "./files/"

type Manuf struct {
	ID        int64
	Title     string
	BodyParts map[int64]BodyPart
}

type BodyPart struct {
	ID               int64
	Title            string
	BodyPartElements map[int64]BodyPartElement
}

type BodyPartElement struct {
	ID         int64
	Title      string
	Required   bool
	Components map[int64]Component
}

type ElBodyPart struct {
	ID    int64
	Title string
}

type Body struct {
	ID     int64
	Title  string
	Manufs map[int64]Manuf
}

type Component struct {
	ID       int64
	Title    string
	Article  string
	Size     sql.NullString
	Addition sql.NullString
	Count    []int
}

type Act struct {
	ID         int64
	BodyPartId int
	Date       time.Time
	Number     string
	Addition   string
	Audit      bool
	File       sql.NullString
	Components map[int64]int
}

type ActData struct {
	Bodies map[int64]Body
}

type ActAddResponse struct {
	Message string
}

func ActsGetPageProcess(w http.ResponseWriter, r *http.Request, router *Router) (*ActData, int, error) {
	db := router.db
	rows, err := db.Query("SELECT * FROM manuf")
	if err != nil {
		log.Println(err.Error())
		return nil, http.StatusInternalServerError, errors.New("internal server error")
	}
	manufs := map[int64]Manuf{}
	for rows.Next() {
		manuf := Manuf{}
		err = rows.Scan(&manuf.ID, &manuf.Title)
		if err != nil {
			log.Println(err.Error())
			return nil, http.StatusInternalServerError, errors.New("internal server error")
		}
		manuf.BodyParts = map[int64]BodyPart{}
		manufs[manuf.ID] = manuf
	}
	rows, err = db.Query("SELECT * FROM body")
	if err != nil {
		log.Println(err.Error())
		return nil, http.StatusInternalServerError, errors.New("internal server error")
	}
	actData := ActData{Bodies: map[int64]Body{}}
	for rows.Next() {
		body := Body{}
		err = rows.Scan(&body.ID, &body.Title)
		if err != nil {
			log.Println(err.Error())
			return nil, http.StatusInternalServerError, errors.New("internal server error")
		}
		body.Manufs = map[int64]Manuf{}
		for key, manuf := range manufs {
			body.Manufs[key] = Manuf{ID: manuf.ID, Title: manuf.Title, BodyParts: map[int64]BodyPart{}}
		}
		actData.Bodies[body.ID] = body
	}
	rows, err = db.Query("SELECT bp.id, bp.title, bp.body_id, bp.manuf_id, bpe.id, bpe.required, ebp.title, c.id, c.article, c.title, c.size, c.addition " +
		"FROM body_part bp " +
		"JOIN body_part_element bpe ON bpe.body_part_id = bp.id " +
		"JOIN el_body_part ebp ON bpe.el_body_part_id = ebp.id " +
		"JOIN bpe_component bpec ON bpec.body_part_element_id = bpe.id " +
		"JOIN component c ON bpec.component_id = c.id " +
		"ORDER BY c.sort ASC")
	if err != nil {
		log.Println(err.Error())
		return nil, http.StatusInternalServerError, errors.New("internal server error")
	}
	var manufactureId int64
	var bodyId int64
	bodyPart := BodyPart{}
	bodyPartElement := BodyPartElement{}
	for rows.Next() {
		component := Component{}
		err = rows.Scan(
			&bodyPart.ID,
			&bodyPart.Title,
			&bodyId,
			&manufactureId,
			&bodyPartElement.ID,
			&bodyPartElement.Required,
			&bodyPartElement.Title,
			&component.ID,
			&component.Article,
			&component.Title,
			&component.Size,
			&component.Addition,
		)
		if err != nil {
			log.Println(err.Error())
			return nil, http.StatusInternalServerError, errors.New("internal server error")
		}
		b := actData.Bodies[bodyId]
		m := b.Manufs[manufactureId]
		var (
			bp  BodyPart
			bpe BodyPartElement
		)
		bp, ok := m.BodyParts[bodyPart.ID]
		if !ok {
			m.BodyParts[bodyPart.ID] = BodyPart{ID: bodyPart.ID, Title: bodyPart.Title, BodyPartElements: map[int64]BodyPartElement{}}
			bp = m.BodyParts[bodyPart.ID]
		}
		bpe, ok = bp.BodyPartElements[bodyPartElement.ID]
		if !ok {
			bp.BodyPartElements[bodyPartElement.ID] = BodyPartElement{
				ID:         bodyPartElement.ID,
				Title:      bodyPartElement.Title,
				Required:   bodyPartElement.Required,
				Components: map[int64]Component{},
			}
			bpe = bp.BodyPartElements[bodyPartElement.ID]
		}
		bpe.Components[component.ID] = component
	}

	return &actData, 0, nil
}

func ActsAddProcess(w http.ResponseWriter, r *http.Request, router *Router) (*ActAddResponse, int, error) {
	db := router.db
	response := ActAddResponse{}
	rollback := func() {
		_, err := db.Exec("ROLLBACK")
		if err != nil {
			log.Println(err.Error())
		}
	}
	err := r.ParseMultipartForm(0)
	if err != nil {
		log.Println(err.Error())
		return nil, http.StatusInternalServerError, errors.New("failed parse form data")
	}
	actBodyPartStr := r.FormValue("body_part_sel")
	actNumber := r.FormValue("actNumber")
	actDateStr := r.FormValue("date")
	actAdditional := r.FormValue("additional")
	actAudit, _ := strconv.ParseBool(r.FormValue("audit"))
	if actBodyPartStr == "" || actNumber == "" || actDateStr == "" {
		log.Println("not found bp, number or date")
		response.Message = "Не все обязательные данные выбраны!"
		return &response, http.StatusOK, nil
	}
	actBodyPart, _ := strconv.Atoi(actBodyPartStr)
	actDate, err := time.Parse("2006-01-02", actDateStr)
	if err != nil {
		log.Println(err.Error())
		return nil, http.StatusInternalServerError, errors.New("invalid date format")
	}
	act := Act{
		BodyPartId: actBodyPart,
		Date:       actDate,
		Number:     actNumber,
		Addition:   actAdditional,
		Audit:      actAudit,
		Components: make(map[int64]int, 5),
	}
	var bpeId int64
	rows, err := db.Query("SELECT id FROM body_part_element")
	if err != nil {
		log.Println(err.Error())
		return nil, http.StatusInternalServerError, errors.New("internal server error")
	}
	for rows.Next() {
		err = rows.Scan(&bpeId)
		if err != nil {
			log.Println(err.Error())
			return nil, http.StatusInternalServerError, errors.New("internal server error")
		}
		if componentId, _ := strconv.Atoi(r.FormValue("Select" + strconv.FormatInt(bpeId, 10))); componentId != 0 {
			act.Components[bpeId] = componentId
		}
	}

	keys := make([]interface{}, 0, len(act.Components))
	for k := range act.Components {
		keys = append(keys, k)
	}
	if len(keys) == 0 {
		log.Println("empty components")
		response.Message = "Данные не выбраны!"
		return &response, http.StatusOK, nil
	}
	keys = append(keys, act.BodyPartId)
	rows, err = db.Query("SELECT DISTINCT body_part_id FROM body_part_element WHERE id IN (? "+strings.Repeat(",?", len(keys)-2)+") AND body_part_id != ?", keys...)
	if err != nil {
		log.Println(err.Error())
		return nil, http.StatusInternalServerError, errors.New("internal server error")
	}
	if rows.Next() {
		log.Println("bp for components not found")
		response.Message = "Неправильно выбраны данные!"
		return &response, http.StatusOK, nil
	}
	if err != nil {
		log.Println(err.Error())
		return nil, http.StatusInternalServerError, errors.New("internal server error")
	}
	if rows.Next() {
		log.Println("components from different bp")
		response.Message = "Неправильно выбраны данные!"
		return &response, http.StatusOK, nil
	}
	if !act.Audit {
		rows, err = db.Query("SELECT id FROM body_part_element WHERE id NOT IN (? "+strings.Repeat(",?", len(keys)-2)+") AND body_part_id = ? AND required = 1", keys...)
		if err != nil {
			log.Println(err.Error())
			return nil, http.StatusInternalServerError, errors.New("internal server error")
		}
		if rows.Next() {
			log.Println("not all required components")
			response.Message = "Не все обязательные данные выбраны!"
			return &response, http.StatusOK, nil
		}
	}
	_, err = db.Exec("START TRANSACTION")
	if err != nil {
		log.Println(err.Error())
		return nil, http.StatusInternalServerError, errors.New("internal server error")
	}
	stmt, err := db.Prepare("INSERT INTO act (addition, date, body_part_id, number, audit, user_id) VALUES (?,?,?,?,?,?)")
	if err != nil {
		log.Println(err.Error())
		defer rollback()
		return nil, http.StatusInternalServerError, errors.New("internal server error")
	}
	res, err := stmt.Exec(act.Addition, act.Date, act.BodyPartId, act.Number, act.Audit, router.getUser(r).ID)
	if err != nil {
		log.Println(err.Error())
		defer rollback()
		return nil, http.StatusInternalServerError, errors.New("internal server error")
	}
	act.ID, err = res.LastInsertId()
	if err != nil {
		log.Println(err.Error())
		defer rollback()
		return nil, http.StatusInternalServerError, errors.New("internal server error")
	}
	file, h, err := r.FormFile("upload")
	switch err {
	case nil:
		defer func(file multipart.File) {
			_ = file.Close()
		}(file)
		filename := strconv.FormatInt(act.ID, 10) + filepath.Ext(h.Filename)
		dst, err := os.Create(fileDir + filename)
		if err != nil {
			log.Println(err.Error())
			defer rollback()
			return nil, http.StatusInternalServerError, errors.New("failed parse form data")
		}
		defer func(dst *os.File) {
			_ = dst.Close()
		}(dst)
		if _, err := io.Copy(dst, file); err != nil {
			log.Println(err.Error())
			defer rollback()
			return nil, http.StatusInternalServerError, errors.New("failed save act file")
		}
		res, err = db.Exec("UPDATE act SET file=? WHERE id=?", filename, act.ID)
		if err != nil {
			log.Println(err.Error())
			defer rollback()
			return nil, http.StatusInternalServerError, errors.New("internal server error")
		}
	case http.ErrMissingFile:
	default:
		log.Println(err.Error())
		defer rollback()
		return nil, http.StatusInternalServerError, errors.New("failed parse form data")
	}

	for _, component := range act.Components {
		res, err = db.Exec("INSERT INTO act_component (act_id, component_id) VALUES (?,?)", act.ID, component)
		if err != nil {
			log.Println(err.Error())
			defer rollback()
			return nil, http.StatusInternalServerError, errors.New("internal server error")
		}
	}
	res, err = db.Exec("COMMIT")
	if err != nil {
		log.Println(err.Error())
		defer rollback()
		return nil, http.StatusInternalServerError, errors.New("internal server error")
	}

	response.Message = "Акт успешно сохранен!"
	return &response, http.StatusOK, nil
}
