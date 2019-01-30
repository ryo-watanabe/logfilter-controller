package fluentbitcfg

import (
  "strings"

  logfilterv1alpha1 "github.com/ryo-watanabe/logfilter-controller/pkg/apis/logfilter/v1alpha1"
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

func MakeFluentbitIgnoreLua(logfilters *logfilterv1alpha1.LogFilterList) map[string]string {
  system_log_filters := ""
  container_log_filters := ""
  pod_log_filters := ""
  for _, filter := range logfilters.Items {
    if filter.Spec.LogKind == "system_log" {
      system_log_filters += "    if record[\"log_name\"] == \"" + filter.Spec.LogName + "\" then\n"
      system_log_filters += filterDefine(filter.Spec.LogName, filter.Spec.Message, filter.Spec.Action)
      system_log_filters += "    end\n"
    } else if filter.Spec.LogKind == "container_log" {
      container_log_filters += "    if record[\"container_name\"] == \"" + filter.Spec.LogName + "\" then\n"
      container_log_filters += filterDefine(filter.Spec.LogName, filter.Spec.Message, filter.Spec.Action)
      container_log_filters += "    end\n"
    } else if filter.Spec.LogKind == "pod_log" {
      pod_log_filters += "    if record[\"kubernetes\"][\"container_name\"] == \"" + filter.Spec.LogName + "\" then\n"
      pod_log_filters += filterDefine(filter.Spec.LogName, filter.Spec.Message, filter.Spec.Action)
      pod_log_filters += "    end\n"
    }
  }
  lua := strings.Replace(fluentbit_ignore_lua, "POD_LOG_FILTERS", pod_log_filters, 1)
  lua = strings.Replace(lua, "CONTAINER_LOG_FILTERS", container_log_filters, 1)
  lua = strings.Replace(lua, "SYSTEM_LOG_FILTERS", system_log_filters, 1)
  return map[string]string{"funcs.lua":lua}
}

func IsValidInFluentbitLua(logfilter *logfilterv1alpha1.LogFilter, currentfluentbitlua string) bool {
  define := filterDefine(logfilter.Spec.LogName, logfilter.Spec.Message, logfilter.Spec.Action)
  if !strings.Contains(currentfluentbitlua, define) {
		return false
	}
	return true
}
