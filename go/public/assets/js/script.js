function dynamicSelect(id1, id2) {	//type=0 по value, type=1 по class

// Сперва необходимо проверить поддержку W3C DOM в браузере

    if (document.getElementById && document.getElementsByTagName) {

// Определение переменных, ссылающихся на списки

        var sel1 = document.getElementById(id1);
        var sel2 = document.getElementById(id2);
        // var sel3 = document.getElementById(id3);

// Клонирование динамического списка

        var clone = sel2.cloneNode(true);
        // var clone2 = sel3.cloneNode(true);
// Определение переменных для клонированных элементов списка

        var clonedOptions = clone.getElementsByTagName("option");
        // var clonedOptions2 = clone2.getElementsByTagName("option");
// Вызов функции собирающей вызываемый список
        hideSelects();
        refreshDynamicSelectOptions(sel1, sel2, id2);
        // refreshDynamicSelectOptions(sel2, sel3, id3, clonedOptions2);
// При изменении выбранного элемента в первом списке: // вызов функции пересобирающей вызываемый список

        sel1.onchange = function() {
            //dynamicSelect(id1, id2, id3);
            hideSelects();
            refreshDynamicSelectOptions(sel1, sel2, id2);
            // refreshDynamicSelectOptions(sel2, sel3, id3, clonedOptions2);
        }
        // sel2.onchange = function() {
        //     //dynamicSelect(id1, id2, id3);
        //     hideSelects();
        //     refreshDynamicSelectOptions(sel2, sel3, id3, clonedOptions2);
        // }
        sel2.onchange = function() {
            //dynamicSelect(id1, id2, id3);
            hideSelects();
            // if(sel1.selectedIndex!==0 && sel2.selectedIndex!==0 && sel3.selectedIndex!==0){
                showSelects(sel2);
            // }
        }
    }
}
function showSelects(sel2){
    var u=0;
    var id_sel=0;
    // var scan_s = document.getElementById("scan_select");
    for(var i=0; ("div"+document.getElementsByName(sel2.options[sel2.selectedIndex].value).item(i)); i++){
        var div = document.getElementsByName("div"+sel2.options[sel2.selectedIndex].value).item(i);
        div.style.display = 'block';
        u++;

        id_sel=div.id.substr(3);
        // if(scan_s.options[id_sel].value!==0){
        //     div = document.getElementById("Select"+id_sel);
        //     div.selectedIndex=scan_s.options[id_sel].value;
        // }
    }
}
function hideSelects(){
    for(var i=1;  (document.getElementById('Div'+i)); i++){
        var div = document.getElementById('Div'+i);
        div.style.display = 'none';
        var sel = document.getElementById('Select'+i);
        sel.selectedIndex = 0;
    }
}

function refreshDynamicSelectOptions(sel1, sel2, id) {
    var sel = document.getElementById(id+'_clone');
    var clone = sel.cloneNode(true);
// Определение переменных для клонированных элементов списка

    var clonedOptions = clone.getElementsByTagName("option");
// Определение переменных для клонированных элементов списка
// Удаление всех элементов динамического списка

    while (sel2.options.length) {
        sel2.remove(0);
    }
    var pattern1 = /( |^)(0)( |$)/;
    var pattern2 = new RegExp("( |^)(" + sel1.options[sel1.selectedIndex].value + ")( |$)");

// Перебор клонированных элементов списка
    var length=clonedOptions.length;
    for (var i = 0; i < length; i++) {

// Если название класса клонированного option эквивалентно значению option первого списка

        if (clonedOptions[i].className.match(pattern1) ||
            clonedOptions[i].className.match(pattern2)) {
// его нужно клонировать в динамически создаваемый список
            sel2.appendChild(clonedOptions[i].cloneNode(true));
        }
    }
}

// Вызов скрипта при загрузке страницы

// window.onload = function() {
//     dynamicSelect("body_part", "body_part_cor", "manuf");
// }
// function scan_sel(){
//     var check = 0;
//     var sel;
//     var scan_sel = document.getElementById('scan_select');
//     for(var i=1;i<=5;i++){
//         for(var u=1;u<=2;u++){
//             check = 0;
//             sel = document.getElementsByName('div'+i+'_'+u);
//             for(var j=0;j<sel.length;j++){
//                 var num = Number.parseInt(sel[j].id.substr(3));
//                 if (scan_sel.options[num].value ==='' && num<34){
//                     check = 1;
//                     break;
//                 }
//             }
//             if (check===0){
//                 var sel1 = document.getElementById('body_part');
//                 var sel2 = document.getElementById('body_part_cor');
//                 var sel3 = document.getElementById('manuf');
//                 if(i<=3){
//                     sel1.selectedIndex=1;
//                     sel1.onchange();
//                     sel2.selectedIndex=i;
//                 }
//                 else{
//                     sel1.selectedIndex=2;
//                     sel1.onchange();
//                     sel2.selectedIndex=i-3;
//                 }
//                 sel2.onchange();
//                 sel3.selectedIndex=(u===1?2:1);
//                 sel3.onchange();
//             }
//         }
//     }
// }
