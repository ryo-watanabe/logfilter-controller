package fluentbitcfg

import (
  "strings"
  "sort"

  corev1 "k8s.io/api/core/v1"
)

const fluentbit_ignore_lua = `function ignore_message(tag, timestamp, record)
  -- pod containers
  if record["kubernetes"] then
POD_LOG_FILTERS
  -- system containers
  elseif record["container_name"] then
CONTAINER_LOG_FILTERS
  -- system logs
  else
SYSTEM_LOG_FILTERS
  end
  return 1, timestamp, record
end

function add_flb_key(tag, timestamp, record)
  new_record = record
  new_record["_flb-key"] = tag
  return 1, timestamp, new_record
end

function add_record(tag, timestamp, record)
  new_record = record
  new_record["container_name"] = string.match( tag, "rke.var.lib.rancher.rke.log.(.*)_%w*.log" )
  return 1, timestamp, new_record
end
`

func filterDefine(log_name, message, action string) string {
  flt :=   "      -- Filter definition to " + action + " " + message + " in " + log_name + "\n"
  if strings.HasPrefix(message, "@startwith:") {
    start := strings.Split(message, ":")[1]
    flt += "      if string.sub(record[\"log\"], 1, string.len(\"" + start + "\")) == \"" + start + "\" then\n"
  } else if strings.HasPrefix(message, "@all") {
    if action == "drop" {
      flt += "      return -1, 0, 0\n"
    } else {
      flt += "      new_record = record\n"
      flt += "      new_record[\"ignore_alerts\"] = \"" + log_name + " - " + message + "\"\n"
      flt += "      return 1, timestamp, new_record\n"
    }
    return flt
  } else {
    flt += "      if string.find(record[\"log\"], \"" + message + "\") then\n"
  }
  if action == "drop" {
    flt += "        return -1, 0, 0\n"
  } else {
    flt += "        new_record = record\n"
    flt += "        new_record[\"ignore_alerts\"] = \"" + log_name + " - " + message + "\"\n"
    flt += "        return 1, timestamp, new_record\n"
  }
  flt +=   "      end\n"
  return flt
}

func pushLogName(log_name string, log_names []string) []string {
  for _, v := range log_names {
    if v == log_name {
      return log_names
    }
  }
  log_names = append(log_names, log_name)
  return log_names
}

func MakeFluentbitIgnoreLua(logfilters *corev1.ConfigMapList) map[string]string {
  // Buffers for each filter definition.
  system_log_filters := map[string]string{}
  container_log_filters := map[string]string{}
  pod_log_filters := map[string]string{}

  // Keys for sort.
  system_log_names := []string{}
  container_log_names := []string{}
  pod_log_names := []string{}

  // Load configmaps
  for _, filter := range logfilters.Items {
    log_kind, kind_ok := filter.Data["log_kind"]
    log_name, name_ok := filter.Data["log_name"]
    message, message_ok := filter.Data["message"]
    action, action_ok := filter.Data["action"]
    if !kind_ok || !name_ok || !message_ok || !action_ok {
      filter.Data["errors"] = "Filter data error"
      continue
    }
    if log_kind == "system_log" {
      system_log_names = pushLogName(log_name, system_log_names)
      system_log_filters[log_name] += filterDefine(log_name, message, action)
    } else if log_kind == "container_log" {
      container_log_names = pushLogName(log_name, container_log_names)
      container_log_filters[log_name] += filterDefine(log_name, message, action)
    } else if log_kind == "pod_log" {
      pod_log_names = pushLogName(log_name, pod_log_names)
      pod_log_filters[log_name] += filterDefine(log_name, message, action)
    }
  }

  // Sort filter difinitions
  system_log := ""
  sort.Strings(system_log_names)
  for _, log_name := range system_log_names {
    system_log += "    if record[\"log_name\"] == \"" + log_name + "\" then\n"
    system_log += system_log_filters[log_name]
    system_log += "    end\n"
  }
  container_log := ""
  sort.Strings(container_log_names)
  for _, log_name := range container_log_names {
    container_log += "    if record[\"container_name\"] == \"" + log_name + "\" then\n"
    container_log += container_log_filters[log_name]
    container_log += "    end\n"
  }
  pod_log := ""
  sort.Strings(pod_log_names)
  for _, log_name := range pod_log_names {
    pod_log += "    if record[\"kubernetes\"][\"container_name\"] == \"" + log_name + "\" then\n"
    pod_log += pod_log_filters[log_name]
    pod_log += "    end\n"
  }

  // Write into lua template
  lua := strings.Replace(fluentbit_ignore_lua, "POD_LOG_FILTERS", pod_log, 1)
  lua = strings.Replace(lua, "CONTAINER_LOG_FILTERS", container_log, 1)
  lua = strings.Replace(lua, "SYSTEM_LOG_FILTERS", system_log, 1)

  return map[string]string{"funcs.lua":lua}
}
