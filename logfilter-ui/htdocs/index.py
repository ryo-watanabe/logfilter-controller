#!/usr/bin/python
# -*- coding: UTF-8 -*-

import commands
import json
import cgi
# import string

# enable debugging
#import cgitb
#cgitb.enable()

# Utilities

def localCommand(com):
    ret_ch = commands.getstatusoutput(com)
    return (ret_ch[0], ret_ch[1])

def renderHead(title):
    print("Content-Type:text/html\n\n")
    print("<html>")
    print("<head>")
    print("<title>" + title + "</title>")
    print("<meta name='viewport' content='width=device-width, initial-scale=1'>")
    # common css
    # print("<link rel='STYLESHEET' href='style.css' type='text/css'>")
    print("<link rel='stylesheet' href='https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap.min.css' integrity='sha384-BVYiiSIFeK1dGmJRAkycuHAHRg32OmUcww7on3RYdg4Va+PmSTsz/K68vbdEjh4u' crossorigin='anonymous'>")
    # common js
    print("<script src='https://ajax.googleapis.com/ajax/libs/jquery/1.12.4/jquery.min.js'></script>")
    print("<script src='https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/js/bootstrap.min.js' integrity='sha384-Tc5IQib027qvyjSMfHjOMaLkfuWVxZxUPnCJA7l2mCWNIpG9mGCD8wGNIcPD7Txa' crossorigin='anonymous'></script>")
    print("<script language='javascript' src='script.js' type='text/javascript'></script>")
    print("</head>")

class HtmlForm():
    action = None
    method = None
    id = None
    params = {}

    def __init__(self, formid, action, method, params=None):
        self.id = formid
        self.action = action
        self.method = method
        if params:
            self.addParams(params)

    def addParam(self, key, value):
        self.params[key] = value

    def addParams(self, params):
        for key, value in params.items():
            self.addParam(key, value)

    def render(self):
        print("<form id='" + self.id + "' action='" + self.action + "' method='" + self.method + "'>")
        for key, value in self.params.items():
            print("<input type='hidden' name='" + key + "' value='" + value + "' />")

#### MAIN ####################

## request params #########
form = cgi.FieldStorage()
req = { 'post_action':'none', 'filter_name':'none', 'action':'none', 'log_kind':'none', 'log_name':'none', 'message':'none' }
for key in req.keys():
    if form.has_key(key):
        req[key] = form[key].value

ret = 0
val = ""
message = ""

if req["post_action"] == "apply":
    cmd = "echo '"
    cmd += '{"apiVersion": "v1","kind": "ConfigMap","metadata": {'
    cmd += '"name": "' + req["filter_name"] + '","labels": {"logfilter.ssl.com/filterdata": "true"}},"data": {'
    cmd += '"action": "' + req["action"] + '",'
    cmd += '"log_kind": "' + req["log_kind"] + '",'
    cmd += '"log_name": "' + req["log_name"] + '",'
    cmd += '"message": "' + req["message"] + '"}}'
    cmd += "' | kubectl --kubeconfig=/kubecfg apply -f -"
    (ret, val) = localCommand(cmd)

if req["post_action"] == "delete":
    (ret, val) = localCommand("kubectl delete cm " + req["filter_name"] + " --kubeconfig=/kubecfg")

params = { 'post_action':'none' }

if ret == 0:
    (ret, val) = localCommand("kubectl get cm -o json --kubeconfig=/kubecfg -l logfilter.ssl.com/filterdata=true")

if ret:
    message = val
else:
    data = json.loads(val)

    add = "<div class='modal fade' role='dialog'><div class='modal-dialog modal-dialog-centered' role='document' tabindex=''-1' role='dialog' aria-labelledby='modal' aria-hidden='true'>"
    add += "<div class='modal-content'><div class='modal-body'>"
    add += "<div class='modal-header'><h2 id='modaltitle'>Add Filter</h2></div>"
    add += "<table class='table'>"
    add += "<tr><th>Filter Name:</th><td><input name='filter_name' class='form-control' value='' id='filternameinput' /></td></tr>"
    add += "<tr><th>Log Kind:</th><td><select name='log_kind' class='form-control'>"
    add += "<option value='system_log'>system_log</option>"
    add += "<option value='container_log'>container_log</option>"
    add += "<option value='pod_log'>pod_log</option>"
    add += "</select></td></tr>"
    add += "<tr><th>Log Name:</th><td><input name='log_name' class='form-control' value='' /></td></tr>"
    add += "<tr><th>Message:</th><td><input name='message' class='form-control' value='' /></td></tr>"
    add += "<tr><th>Action:</th><td><select name='action' class='form-control'>"
    add += "<option value='ignore'>ignore</option>"
    add += "<option value='drop'>drop</option>"
    add += "</select></td></tr>"
    add += "</table>"
    add += "<p class='text-right'><a href='javascript:applyFilter()' class='btn btn-primary' id='modalapplybtn'>Add</a>"
    add += " <button type='button' class='btn btn-secondary' data-dismiss='modal'>Close</button></p>"
    add += "</div></div></div></div>"

    logfilters = "<table class='table'>"
    lf = "<thead><tr>"
    lf += "<th>Filter Name</th>"
    lf += "<th>Kind</th>"
    lf += "<th>Log Name</th>"
    lf += "<th>Ignore Message</th>"
    lf += "<th>action</th>"
    lf += "<th></th>"
    lf += "</tr></thead>"
    lf += "<tbody>"
    logfilters += lf
    for item in data["items"]:
        lf = "<tr>"
        lf += "<td>" + item["metadata"]["name"] + "</td>"
        lf += "<td>" + item["data"]["log_kind"] + "</td>"
        lf += "<td>" + item["data"]["log_name"] + "</td>"
        lf += "<td>" + cgi.escape(item["data"]["message"]).encode('ascii', 'xmlcharrefreplace') + "</td>"
        lf += "<td>" + item["data"]["action"] + "</td>"
        lf += "<td><a href='javascript:editFilter(" + \
              '"' + item["metadata"]["name"] + '",' + \
              '"' + item["data"]["log_kind"] + '",' + \
              '"' + item["data"]["log_name"] + '",' + \
              '"' + cgi.escape(item["data"]["message"]).encode('ascii', 'xmlcharrefreplace') + '",' + \
              '"' + item["data"]["action"] + '"' + ")'><span class='glyphicon glyphicon-pencil'></span></a></td>"
        lf += "<td><a href='javascript:deleteFilter(" + '"' + item["metadata"]["name"] + '"' + ")'><span class='glyphicon glyphicon-trash'></span></a></td>"
        lf += "</tr>"
        logfilters += lf
    logfilters += "</tbody></table>"

# Render HTML

renderHead("Logfilter Manager")
print("<body><div class='container'>")
if ret:
    print("Return Code : " + str(ret) + "<br>")
print("<h2>Applied Log Filters : </h2>")
print("<h2 class='text-right'><a href='javascript:addFilter()'><span class='glyphicon glyphicon-plus'></span></a></h2>")
HtmlForm("form1", "index.py", "POST", params).render()
print("<div class='form-group'>")
if message:
    print("Output: " + message)
else:
    print(add + logfilters)
print("</div></form>")
print("</div></body>")
print("</html>")
