function applyFilter() {
    form = document.getElementById('form1');
    form.post_action.value = "apply";
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

function modalShow() {
  jQuery('.modal').modal('show');
}

function addFilter() {
  form = document.getElementById('form1');
  form.filter_name.value = "";
  form.log_kind.value = "system_log";
  form.log_name.value = "";
  form.message.value = "";
  form.action.value = "ignore";
  document.getElementById('modalapplybtn').innerHTML = "Add"
  document.getElementById('modaltitle').innerHTML = "Add Filter"
  document.getElementById('filternameinput').readOnly = false;
  jQuery('.modal').modal('show');
}

function editFilter(filter_name, log_kind, log_name, message, action) {
  form = document.getElementById('form1');
  form.filter_name.value = filter_name;
  form.log_kind.value = log_kind;
  form.log_name.value = log_name;
  form.message.value = message;
  form.action.value = action;
  document.getElementById('modalapplybtn').innerHTML = "Apply"
  document.getElementById('modaltitle').innerHTML = "Edit Filter"
  document.getElementById('filternameinput').readOnly = true;
  jQuery('.modal').modal('show');
}
