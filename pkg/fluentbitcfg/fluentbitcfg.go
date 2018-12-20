package fluentbitcfg

import (
  "strings"
	"strconv"

  logfiltersv1alpha1 "k8s.io/sample-controller/pkg/apis/logfilters/v1alpha1"
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
  flt :=   "    if string.find(record[\"log\"], \"" + message + "\") then\n"
  if strings.HasPrefix(message, "@startwith:") {
    start := strings.Split(message, ":", 2)[1]
    flt := "    if string.sub(record[\"log\"], 1, string.len(\"" + start + "\")) == \"" + start + "\" then\n"
  }
  if action == "drop" {
    flt += "      return -1, 0, 0\n"
  } else {
    flt += "      new_record = record\n"
    flt += "      new_record[\"ignore_alerts\"] = \"ignore - " + message + "\"\n"
    flt += "      return 1, timestamp, new_record\n"
  }
  flt +=   "    end\n"
  return flt
}

func MakeFluentbitIgnoreLua(logfilters *logfiltersv1alpha1.LogfilterList) map[string]string {
  system_log_filters := ""
  container_log_filters := ""
  pod_log_filters := ""
  for _, filter := range logfilters.Items {
    if filter.Spec.LogKind == "system_log" {
      system_log_filters += "  if record[\"log_name\"] == \"" + flilter.Spec.LogName + "\" then\n"
      system_log_filters += filterDefine(flilter.Spec.LogName, flilter.Spec.Message, flilter.Spec.Action)
      system_log_filters += "  end\n"
    } else if filter.Spec.LogKind == "container_log" {
      container_log_filters += "  if record[\"container_name\"] == \"" + flilter.Spec.LogName + "\" then\n"
      container_log_filters += filterDefine(flilter.Spec.LogName, flilter.Spec.Message, flilter.Spec.Action)
      container_log_filters += "  end\n"
    } else if filter.Spec.LogKind == "pod_log" {
      pod_log_filters += "  if record[\"kubernetes\"][\"container_name\"] == \"" + flilter.Spec.LogName + "\" then\n"
      pod_log_filters += filterDefine(flilter.Spec.LogName, flilter.Spec.Message, flilter.Spec.Action)
      pod_log_filters += "  end\n"
    }
  }
  lua := strings.Replace(fluentbit_ignore_lua, "POD_LOG_FILTERS", pod_log_filters)
  lua = strings.Replace(lua, "CONTAINER_LOG_FILTERS", container_log_filters)
  lua = strings.Replace(lua, "SYSTEM_LOG_FILTERS", system_log_filters)
  return map[string]string{"funcs.lua":lua}
}

func IsValicInFluentbitLua(logfilter *logfiltersv1alpha1, currentfluentbitlua string) bool {
  define := filterDefine(logflilter.Spec.LogName, logflilter.Spec.Message, logflilter.Spec.Action)
  if !strings.Contains(currentfluentbitlua, define) {
		return false
	}
	return true
}
