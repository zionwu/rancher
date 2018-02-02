package generator

var ClusterTemplate = `{{ if ne .clusterTarget.CurrentTarget "none" }}
<source>
   @type  tail
   path  /var/log/containers/*.log
   pos_file  /fluentd/etc/log/fluentd-cluster-logging.pos
   time_format  %Y-%m-%dT%H:%M:%S
   tag  cluster.*
   format  json
   read_from_head  true
</source>

<filter  cluster.**>
   @type  kubernetes_metadata
   merge_json_log  true
   preserve_json_log  true
</filter>

<filter cluster.**>
  @type record_transformer
  <record>
    tag ${tag}
    {{range $k, $val := .clusterTarget.OutputTags -}}
    {{$k}} {{$val}}
    {{end -}}
  </record>
</filter>

<match  cluster.**> 
    {{ if eq .clusterTarget.CurrentTarget "embedded"}}
    @type elasticsearch
    include_tag_key  true
    hosts "elasticsearch.cattle-logging:9200"
    reload_connections "true"
    logstash_prefix {{.clusterTarget.ClusterName}}
    logstash_format true
    logstash_dateformat  {{.clusterTarget.WrapEmbedded.DateFormat}}
    type_name  "container_log"
    reload_connections false
    {{end -}}

    {{ if eq .clusterTarget.CurrentTarget "elasticsearch"}}
    @type elasticsearch
    include_tag_key  true
    hosts {{.clusterTarget.ElasticsearchConfig.Endpoint}}
    reload_connections "true"
    logstash_prefix "{{.clusterTarget.ElasticsearchConfig.IndexPrefix}}"
    logstash_format true
    logstash_dateformat  {{.clusterTarget.WrapElasticsearch.DateFormat}}
    type_name  "container_log"
    reload_connections false
    {{end -}}

    {{ if eq .clusterTarget.CurrentTarget "splunk"}}
    @type splunk-http-eventcollector
    server  {{.clusterTarget.WrapSplunk.Server}}
    all_items true
    protocol {{.clusterTarget.WrapSplunk.Scheme}}
    sourcetype {{.clusterTarget.SplunkConfig.Source}}
    token {{.clusterTarget.SplunkConfig.Token}}
    format json
    reload_connections "true"
    {{end -}}

    {{ if eq .clusterTarget.CurrentTarget "kafka"}}
    @type kafka_buffered
    {{ if .clusterTarget.KafkaConfig.ZookeeperEndpoint }}
    zookeeper {{.clusterTarget.WrapKafkaConfig.Zookeeper}}
    {{else}}
    brokers {{.clusterTarget.KafkaConfig.BrokerEndpoints}}
    {{end}}
    default_topic {{.clusterTarget.KafkaConfig.Topic}}
    output_data_type  "json"
    output_include_tag true
    output_include_time true
    # get_kafka_client_log  true
    max_send_retries  3
    {{end -}}

    {{ if eq .clusterTarget.CurrentTarget "syslog"}}
    @type remote_syslog
    host {{.clusterTarget.WrapSyslog.Host}}
    port {{.clusterTarget.WrapSyslog.Port}}
    severity {{.clusterTarget.SyslogConfig.Severity}}
    program {{.clusterTarget.SyslogConfig.Program}}
    {{end -}}

    flush_interval 2s
    buffer_type file
    buffer_path /fluentd/etc/buffer/cluster.buffer
    buffer_queue_limit 128
    buffer_chunk_limit 256m
    max_retry_wait 30
    disable_retry_limit
    num_threads 8
    slow_flush_log_threshold 40.0
</match>

{{end -}}
`
