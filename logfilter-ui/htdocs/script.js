function addFilter() {
    form = document.getElementById('form1');
    form.post_action.value = "add";
    form.submit();
}

function deleteFilter(filter_name) {
    if (window.confirm("Deleting logfilter " + filter_name + ". OK?")) {
        form = document.getElementById('form1');
        form.post_action.value = "delete";
        form.filter_name.value = filter_name;
        form.submit();
    }
}

//function applyFilter() {
//    if (window.confirm("Applying logfilter. OK?")) {
//        form = document.getElementById('form1');
//        form.post_action.value = "apply";
//        form.submit();
//    }
//}
