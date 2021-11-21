package main

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"
)

func StatisticsProcess(w http.ResponseWriter, r *http.Request, router *Router) (*map[string]interface{}, int, error) {
	db := router.db
	query := r.URL.Query()
	dateMinStr := query.Get("date_min")
	dateMaxStr := query.Get("date_max")
	bodyPartStr := query.Get("body_part")
	var dateMin, dateMax time.Time
	var err error
	if dateMinStr == "" {
		dateMin = time.Now().AddDate(0, -1, 0)
	} else {
		dateMin, err = time.Parse("2006-01-02", dateMinStr)
		if err != nil {
			log.Println(err.Error())
			return nil, http.StatusInternalServerError, errors.New("failed parse data_min")
		}
	}
	if dateMaxStr == "" {
		dateMax = time.Now()
	} else {
		dateMax, err = time.Parse("2006-01-02", dateMaxStr)
		if err != nil {
			log.Println(err.Error())
			return nil, http.StatusInternalServerError, errors.New("failed parse data_max")
		}
	}
	rows, err := db.Query("SELECT bp.id, CONCAT(b.title, ' ', bp.title, ' ', m.title) " +
		"FROM body_part bp JOIN manuf m ON bp.manuf_id = m.id " +
		"JOIN body b ON bp.body_id = b.id")
	if err != nil {
		log.Println(err.Error())
		return nil, http.StatusInternalServerError, errors.New("internal server error")
	}
	bodyParts := make(map[int64]BodyPart, 10)
	bodyPart := BodyPart{}
	for rows.Next() {
		err = rows.Scan(&bodyPart.ID, &bodyPart.Title)
		if err != nil {
			log.Println(err.Error())
			return nil, http.StatusInternalServerError, errors.New("internal server error")
		}
		bodyParts[bodyPart.ID] = bodyPart
	}
	bodyPartId, _ := strconv.Atoi(bodyPartStr)
	queryStr := "SELECT a.id, a.addition, a.date, a.number, a.file, a.audit, a.body_part_id, c.id, c.article, " +
		"c.title, c.addition, c.size FROM act a " +
		"JOIN act_component ac ON ac.act_id = a.id JOIN component c ON ac.component_id = c.id " +
		"JOIN body_part bp ON a.body_part_id = bp.id JOIN manuf m ON bp.manuf_id = m.id " +
		"JOIN body b ON bp.body_id = b.id WHERE a.date BETWEEN ? AND ? "
	args := []interface{}{dateMin, dateMax}
	if bodyPartId != 0 {
		queryStr = queryStr + "AND a.body_part_id = ? "
		args = append(args, bodyPartId)
	}
	rows, err = db.Query(queryStr+"ORDER BY a.date ASC", args...)
	if err != nil {
		log.Println(err.Error())
		return nil, http.StatusInternalServerError, errors.New("internal server error")
	}
	acts := map[int64]Act{}
	act := Act{}
	components := map[int64]Component{}
	component := Component{}
	years := make(map[string]string, 10)
	for rows.Next() {
		err = rows.Scan(&act.ID, &act.Addition, &act.Date, &act.Number, &act.File, &act.Audit, &act.BodyPartId, &component.ID,
			&component.Article, &component.Title, &component.Addition, &component.Size)
		if err != nil {
			log.Println(err.Error())
			return nil, http.StatusInternalServerError, errors.New("internal server error")
		}
		actYear := act.Date.Format("2006")
		if _, ok := years[actYear]; !ok {
			years[actYear] = actYear
		}
		if _, ok := acts[act.ID]; !ok {
			act.Components = map[int64]int{}
			acts[act.ID] = act
		}
		if _, ok := components[component.ID]; !ok {
			components[component.ID] = component
		}
		acts[act.ID].Components[component.ID] = int(component.ID)
	}
	params := map[string]interface{}{
		"dateMin":  dateMin.Format("2006-01-02"),
		"dateMax":  dateMax.Format("2006-01-02"),
		"bodyPart": int64(bodyPartId),
		"years":    years,
	}
	data := map[string]interface{}{"acts": &acts, "components": &components, "bodyParts": &bodyParts, "params": &params}
	return &data, http.StatusOK, nil
}
func StatisticsComponentsProcess(w http.ResponseWriter, r *http.Request, router *Router) (*map[string]map[string]map[int64]Component, int, error) {
	db := router.db
	query := r.URL.Query()
	bodyPartStr := query.Get("body_part")
	yearStr := query.Get("year")
	var err error
	if yearStr == "" {
		log.Println(err.Error())
		return nil, http.StatusInternalServerError, errors.New("empty year param")
	}
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		log.Println(err.Error())
		return nil, http.StatusInternalServerError, errors.New("failed parse year")
	}
	bodyPart, err := strconv.Atoi(bodyPartStr)
	if err != nil {
		log.Println(err.Error())
		return nil, http.StatusInternalServerError, errors.New("failed parse bodyPart")
	}
	queryStr := "SELECT DISTINCT c.id, c.article, c.title, c.size, c.addition, m.title, ebp.title, COUNT(DISTINCT ac.id) " +
		"FROM component c " +
		"JOIN act_component ac ON ac.component_id = c.id " +
		"JOIN act a ON ac.act_id = a.id " +
		"JOIN body_part bp ON a.body_part_id = bp.id " +
		"JOIN bpe_component bpec ON bpec.component_id = c.id " +
		"JOIN body_part_element bpe ON bpec.body_part_element_id = bpe.id " +
		"JOIN el_body_part ebp ON ebp.id = bpe.el_body_part_id " +
		"JOIN manuf m ON bp.manuf_id = m.id " +
		"WHERE YEAR(a.date) = ? AND MONTH(a.date) = ? "
	queryParams := make([]interface{}, 2, 3)
	queryParams[0] = year
	if bodyPart != 0 {
		queryStr += "AND bp.id = ? "
		queryParams = append(queryParams, bodyPart)
	}
	queryStr += "GROUP BY c.id"
	component := Component{}
	manuf := Manuf{}
	elBodyPart := ElBodyPart{}
	componentCount := 0
	data := make(map[string]map[string]map[int64]Component)
	for month := 1; month <= 12; month++ {
		queryParams[1] = month
		rows, err := db.Query(queryStr, queryParams...)
		if err != nil {
			log.Println(err.Error())
			return nil, http.StatusInternalServerError, errors.New("internal server error")
		}
		for rows.Next() {
			err = rows.Scan(
				&component.ID,
				&component.Article,
				&component.Title,
				&component.Size,
				&component.Addition,
				&manuf.Title,
				&elBodyPart.Title,
				&componentCount,
			)
			if err != nil {
				log.Println(err.Error())
				return nil, http.StatusInternalServerError, errors.New("internal server error")
			}
			if _, ok := data[manuf.Title]; !ok {
				data[manuf.Title] = make(map[string]map[int64]Component)
			}
			ebps := data[manuf.Title]
			if _, ok := ebps[elBodyPart.Title]; !ok {
				ebps[elBodyPart.Title] = make(map[int64]Component)
			}
			components := ebps[elBodyPart.Title]
			if _, ok := components[component.ID]; !ok {
				components[component.ID] = Component{
					ID:       component.ID,
					Title:    component.Title,
					Article:  component.Article,
					Size:     component.Size,
					Addition: component.Addition,
					Count:    make([]int, 13),
				}
			}
			components[component.ID].Count[month] = componentCount
			components[component.ID].Count[0] += componentCount
			ebps[elBodyPart.Title] = components
			data[manuf.Title] = ebps
		}
	}
	return &data, http.StatusOK, nil
}
