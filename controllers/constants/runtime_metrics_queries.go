/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package constants

const (
	//Caikit Standalone
	CaikitData = `{
        "config": [
            {
                "title": "Number of requests",
                "type": "REQUEST_COUNT",
                "queries": [
                    {
                        "title": "Number of successful incoming requests",
                        "query": "round(sum(increase(predict_rpc_count_total{namespace='${NAMESPACE}',code='OK',model_id='${MODEL_NAME}'}[${RATE_INTERVAL}])))"
                    },
                    {
                        "title": "Number of failed incoming requests",
                        "query": "round(sum(increase(predict_rpc_count_total{namespace='${NAMESPACE}',code!='OK',model_id='${MODEL_NAME}'}[${RATE_INTERVAL}])))"
                    }
                ]
            },
            {
                "title": "Average response time (ms)",
                "type": "MEAN_LATENCY",
                "queries": [
                    {
                        "title": "Average inference latency",
                        "query": "sum by (model_id) (rate(predict_caikit_library_duration_seconds_sum{namespace='${NAMESPACE}',model_id='${MODEL_NAME}'}[1m])) / sum by (model_id) (rate(predict_caikit_library_duration_seconds_count{namespace='${NAMESPACE}',model_id='${MODEL_NAME}'}[${RATE_INTERVAL}]))"
                    },
                    {
                        "title": "Average e2e latency",
                        "query": "sum by (model_id) (rate(caikit_core_load_model_duration_seconds_sum{namespace='${NAMESPACE}',model_id='${MODEL_NAME}'}[1m]) + rate(predict_caikit_library_duration_seconds_sum{namespace='${NAMESPACE}',model_id='${MODEL_NAME}'}[1m])) / sum by (model_id) (rate(caikit_core_load_model_duration_seconds_count{namespace='${NAMESPACE}',model_id='${MODEL_NAME}'}[${RATE_INTERVAL}]) + rate(predict_caikit_library_duration_seconds_count{namespace='${NAMESPACE}',model_id='${MODEL_NAME}'}[${RATE_INTERVAL}]))"
                    }
                ]
            },
            {
                "title": "CPU utilization %",
                "type": "CPU_USAGE",
                "queries": [
                    {
                        "title": "CPU usage",
                        "query": "sum(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{namespace='${NAMESPACE}'} * on(namespace,pod) group_left(workload, workload_type) namespace_workload_pod:kube_pod_owner:relabel{namespace='${NAMESPACE}', workload=~'${MODEL_NAME}-predictor-.*', workload_type=~'deployment'}) by (pod)"
                    }
                ]
            },
            {
                "title": "Memory utilization %",
                "type": "MEMORY_USAGE",
                "queries": [
                    {
                        "title": "Memory usage",
                        "query":  "sum(container_memory_working_set_bytes{namespace='${NAMESPACE}', pod=~'${MODEL_NAME}-predictor-.*'}) by (pod)"
                    }
                ]
            }
        ]
    }`

	//OpenVino Model Server
	OvmsData = `{
        "config": [
            {
                "title": "Number of requests",
                "type": "REQUEST_COUNT",
                "queries": [
                    {
                        "title": "Number of successful incoming requests",
                        "query": "round(sum(increase(ovms_requests_success{namespace='${NAMESPACE}',name='${MODEL_NAME}'}[${RATE_INTERVAL}])))"
                    },
                    {
                        "title": "Number of failed incoming requests",
                        "query": "round(sum(increase(ovms_requests_fail{namespace='${NAMESPACE}',name='${MODEL_NAME}'}[${RATE_INTERVAL}])))"
                    }
                ]
            },
            {
                "title": "Average response time (ms)",
                "type": "MEAN_LATENCY",
                "queries": [
                    {
                        "title": "Average inference latency",
                        "query": "sum by (name) (rate(ovms_inference_time_us_sum{namespace='${NAMESPACE}', name='${MODEL_NAME}'}[1m])) / sum by (name) (rate(ovms_inference_time_us_count{namespace='${NAMESPACE}', name='${MODEL_NAME}'}[${RATE_INTERVAL}]))"
                    },
                    {
                        "title": "Average e2e latency",
                        "query": "sum by (name) (rate(ovms_request_time_us_sum{name='${MODEL_NAME}'}[1m])) / sum by (name) (rate(ovms_request_time_us_count{name='${MODEL_NAME}'}[${RATE_INTERVAL}]))"
                    }
                ]
            },
            {
                "title": "CPU utilization %",
                "type": "CPU_USAGE",
                "queries": [
                    {
                        "title": "CPU usage",
                        "query": "sum(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{namespace='${NAMESPACE}'} * on(namespace,pod) group_left(workload, workload_type) namespace_workload_pod:kube_pod_owner:relabel{namespace='${NAMESPACE}', workload=~'${MODEL_NAME}-predictor-.*', workload_type=~'deployment'}) by (pod)"
                    }
                ]
            },
            {
                "title": "Memory utilization %",
                "type": "MEMORY_USAGE",
                "queries": [
                    {
                        "title": "Memory usage",
                        "query": "sum(container_memory_working_set_bytes{namespace='${NAMESPACE}', pod=~'${MODEL_NAME}-predictor-.*'}) by (pod)"
                    }
                ]
            }
        ]
    }`

	// Caikit + TGIS
	TgisData = `{
        "config": [
            {
                "title": "Number of requests",
                "type": "REQUEST_COUNT",
                "queries": [
                    {
                        "title": "Number of successful incoming requests",
                        "query": "round(sum(increase(tgi_request_success{namespace='${NAMESPACE}', pod=~'${MODEL_NAME}-predictor-.*'}[${RATE_INTERVAL}])))"
                    },
                    {
                        "title": "Number of failed incoming requests",
                        "query": "round(sum(increase(tgi_request_failure{namespace='${NAMESPACE}', pod=~'${MODEL_NAME}-predictor-.*'}[${RATE_INTERVAL}])))"
                    }
                ]
            },
            {
                "title": "Average response time (ms)",
                "type": "MEAN_LATENCY",
                "queries": [
                    {
                        "title": "Average inference latency",
                        "query": "sum by (pod) (rate(tgi_request_inference_duration_sum{namespace='${NAMESPACE}', pod=~'${MODEL_NAME}-predictor-.*'}[${RATE_INTERVAL}])) / sum by (pod) (rate(tgi_request_inference_duration_count{namespace='${NAMESPACE}', pod=~'${MODEL_NAME}-predictor-.*'}[${RATE_INTERVAL}]))  "
                    },
                    {
                        "title": "Average e2e latency",
                        "query": "sum by (pod) (rate(tgi_request_duration_sum{namespace='${NAMESPACE}', pod=~'${MODEL_NAME}-predictor-.*'}[${RATE_INTERVAL}])) / sum by (pod) (rate(tgi_request_duration_count{namespace='${NAMESPACE}', pod=~'${MODEL_NAME}-predictor-.*'}[${RATE_INTERVAL}]))"
                    }
                ]
            },
            {
                "title": "CPU utilization %",
                "type": "CPU_USAGE",
                "queries": [
                    {
                        "title": "CPU usage",
                        "query": "sum(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{namespace='${NAMESPACE}'} * on(namespace,pod) group_left(workload, workload_type) namespace_workload_pod:kube_pod_owner:relabel{namespace='${NAMESPACE}', workload=~'${MODEL_NAME}-predictor-.*', workload_type=~'deployment'}) by (pod)"
                    }
                ]
            },
            {
                "title": "Memory utilization %",
                "type": "MEMORY_USAGE",
                "queries": [
                    {
                        "title": "Memory usage",
                        "query": "sum(container_memory_working_set_bytes{namespace='${NAMESPACE}', pod=~'${MODEL_NAME}-predictor-.*'}) by (pod)"
                    }
                ]
            }
        ]
    }`

	//vLLM
	VllmData = `{
        "config": [
            {
                "title": "Number of requests",
                "type": "REQUEST_COUNT",
                "queries": [
                    {
                        "title": "Number of successful incoming requests",
                        "query": "round(sum(increase(vllm:request_success_total{namespace='${NAMESPACE}',model_name='${model_name}'}[${RATE_INTERVAL}])))"
                    }
                ]
            },
            {
                "title": "Average response time (ms)",
                "type": "MEAN_LATENCY",
                "queries": [
                    {
                        "title": "Average e2e latency",
                        "query": "histogram_quantile(0.5, sum(rate(vllm:e2e_request_latency_seconds_bucket{namespace='${NAMESPACE}', model_name='${MODEL_NAME}'}[${RATE_INTERVAL}])) by (le, model_name))"
                    }
                ]
            },
            {
                "title": "CPU utilization %",
                "type": "CPU_USAGE",
                "queries": [
                    {
                        "title": "CPU usage",
                        "query": "sum(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{namespace='${NAMESPACE}'} * on(namespace,pod) group_left(workload, workload_type) namespace_workload_pod:kube_pod_owner:relabel{namespace='${NAMESPACE}', workload=~'${MODEL_NAME}-predictor-.*', workload_type=~'deployment'}) by (pod)"
                    }
                ]
            },
            {
                "title": "Memory utilization %",
                "type": "MEMORY_USAGE",
                "queries": [
                    {
                        "title": "Memory usage",
                        "query":  "sum(container_memory_working_set_bytes{namespace='${NAMESPACE}', pod=~'${MODEL_NAME}-predictor-.*'}) by (pod)"
                    }
                ]
            }
        ]
    }`
)
