package raybuilder

import (
	"context"
	"fmt"
	"os"

	aiApi "github.com/splunk/splunk-ai-operator/api/v1"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const DefaultApplicationsYaml = `applications:
    - name: uae_large
      args:
        deployment_configs:
          EncoderDeployment:
            env_options_override:
              prod:
                autoscaling_config:
                  min_replicas: 1
            options:
              autoscaling_config:
                max_replicas: 12
                min_replicas: 3
                target_ongoing_requests: 3
              ray_actor_options:
                num_gpus: 0.05
        deployment_type: encoder_deployment
        model_definition:
          model_id: uae_large
          model_loader:
            s3_artifact:
              s3_key_prefix: model_artifacts/uae-large
          model_type: sentence_transformer
              
`

const TestDefaultApplicationsYaml = `applications:
    - name: deepseek_r1_distill_llama_8b
      args:
        deployment_configs:
          LLMDeployment:
            options:
              ray_actor_options:
                num_gpus: 1
        deployment_type: vllm_text_gen_model_deployment
        model_definition:
          model_config:
            engine_args:
              tensor_parallel_size: 1
          model_id: deepseek_r1_distill_llama_8b
          model_loader:
            hugging_face_id:
              hf_model_id: deepseek-ai/DeepSeek-R1-Distill-Llama-8B
      runtime_env:
        env_vars:
          VLLM_WORKER_MULTIPROC_METHOD: spawn

    - name: deepseek_r1_distill_llama_70b_awq
      args:
        deployment_configs:
          LLMDeployment:
            options:
              autoscaling_config:
                target_ongoing_requests: 3
              max_ongoing_requests: 4
              ray_actor_options:
                num_gpus: 2
        deployment_type: vllm_text_gen_model_deployment
        model_definition:
          model_config:
            engine_args:
              gpu_memory_utilization: 0.95
              tensor_parallel_size: 2
          model_id: deepseek_r1_distill_llama_70b_awq
          model_loader:
            hugging_face_id:
              hf_model_id: casperhansen/deepseek-r1-distill-llama-70b-awq
      runtime_env:
        env_vars:
          VLLM_WORKER_MULTIPROC_METHOD: spawn

    - name: uae_large
      args:
        deployment_configs:
          EncoderDeployment:
            env_options_override:
              prod:
                autoscaling_config:
                  min_replicas: 1
            options:
              autoscaling_config:
                max_replicas: 12
                min_replicas: 3
                target_ongoing_requests: 3
              ray_actor_options:
                num_gpus: 0.05
        deployment_type: encoder_deployment
        model_definition:
          model_id: uae_large
          model_loader:
            s3_artifact:
              s3_key_prefix: model_artifacts/uae-large
          model_type: sentence_transformer

    - name: llava15_13b
      args:
        custom_deployment_import_path: llava15_13b:VLLMGenerativeModelDeployment
        deployment_configs:
          VLLMGenerativeModelDeployment:
            options:
              autoscaling_config:
                max_replicas: 1
                min_replicas: 0
              ray_actor_options:
                num_gpus: 1
        deployment_type: custom_deployment
      runtime_env:
        env_vars:
          VLLM_WORKER_MULTIPROC_METHOD: spawn
        pip:
        - python-multipart

    - name: mistral
      args:
        deployment_configs:
          LLMDeployment:
            options:
              ray_actor_options:
                num_gpus: 1
        deployment_type: vllm_text_gen_model_deployment
        model_definition:
          model_config:
            engine_args:
              tensor_parallel_size: 1
          model_id: mistral
          model_loader:
            s3_artifact:
              s3_key_prefix: model_artifacts/mistral
        tokenizer_definition:
          model_id: mistral
          model_loader:
            s3_artifact:
              artifacts_list:
              - tokenizer_config.json
              - tokenizer.model
              s3_key_prefix: model_artifacts/mistral
      runtime_env:
        env_vars:
          VLLM_WORKER_MULTIPROC_METHOD: spawn

    - name: all_minilm_l6_v2
      args:
        deployment_configs:
          EncoderDeployment:
            options:
              autoscaling_config:
                max_replicas: 12
                min_replicas: 1
                target_ongoing_requests: 3
              ray_actor_options:
                num_gpus: 0.01
        deployment_type: encoder_deployment
        model_definition:
          model_id: all_minilm_l6_v2
          model_loader:
            s3_artifact:
              s3_key_prefix: model_artifacts/all-minilm-l6-v2
          model_type: sentence_transformer

    - name: bi_encoder
      args:
        deployment_type: encoder_deployment
        model_definition:
          model_id: bi_encoder
          model_loader:
            s3_artifact:
              s3_key_prefix: model_artifacts/bi-encoder
          model_type: sentence_transformer

    - name: mbart_translator
      args:
        custom_deployment_import_path: mbart_translator:MbartTranslatorDeployment
        deployment_configs:
          MbartTranslatorDeployment:
            options:
              ray_actor_options:
                num_gpus: 0.1
        deployment_type: custom_deployment

    - name: spacy_di
      args:
        custom_deployment_import_path: spacy_di:SpacyDiDeployment
        deployment_configs:
          SpacyDiDeployment:
            options:
              ray_actor_options:
                num_gpus: 0.01
        deployment_type: custom_deployment

    - name: phi4
      args:
        deployment_configs:
          LLMDeployment:
            options:
              ray_actor_options:
                num_gpus: 1
        deployment_type: vllm_text_gen_model_deployment
        model_definition:
          model_config:
            engine_args:
              tensor_parallel_size: 1
          model_id: phi4
          model_loader:
            hugging_face_id:
              hf_model_id: NyxKrage/Microsoft_Phi-4
      runtime_env:
        env_vars:
          VLLM_WORKER_MULTIPROC_METHOD: spawn

    - name: xlm_roberta_language_classifier
      args:
        deployment_type: classifier_deployment
        model_definition:
          model_id: xlm_roberta_language_classifier
          model_loader:
            s3_artifact:
              s3_key_prefix: model_artifacts/xlm-roberta-language-classifier

    - name: prompt_injection_tfidf
      args:
        custom_deployment_import_path: prompt_injection_tfidf:PromptInjectionTfidfDeployment
        deployment_type: custom_deployment

    - name: cross_encoder
      args:
        deployment_configs:
          EncoderDeployment:
            options:
              ray_actor_options:
                num_gpus: 0.01
        deployment_type: encoder_deployment
        model_definition:
          model_id: cross_encoder
          model_loader:
            s3_artifact:
              s3_key_prefix: model_artifacts/cross-encoder
          model_type: sentence_transformer_cross_encoder

    - name: pii_classifier
      args:
        custom_deployment_import_path: pii_classifier:PIIClassifierDeployment
        deployment_configs:
          PIIClassifierDeployment:
            options:
              ray_actor_options:
                num_cpus: 4
        deployment_type: custom_deployment

    - name: llama31_instruct
      args:
        deployment_configs:
          LLMDeployment:
            env_options_override:
              prod:
                autoscaling_config:
                  min_replicas: 1
            gpu_type_options_override:
              A10G:
                ray_actor_options:
                  num_gpus: 2
              L40S:
                ray_actor_options:
                  num_gpus: 1
              T4:
                ray_actor_options:
                  num_gpus: 4
            options:
              autoscaling_config:
                min_replicas: 2
        deployment_type: vllm_text_gen_model_deployment
        model_definition:
          gpu_type_model_config_override:
            A10G:
              engine_args:
                max_model_len: 14190
                tensor_parallel_size: 2
            L40S:
              engine_args:
                max_model_len: 14190
                tensor_parallel_size: 1
            T4:
              engine_args:
                dtype: half
                max_model_len: 14190
                tensor_parallel_size: 4
          model_id: llama31_instruct
          model_loader:
            s3_artifact:
              s3_key_prefix: model_artifacts/llama31-8b-instruct
        tokenizer_definition:
          model_id: llama31_instruct
          model_loader:
            s3_artifact:
              artifacts_list:
              - tokenizer_config.json
              - tokenizer.json
              s3_key_prefix: model_artifacts/llama31-8b-instruct
      runtime_env:
        env_vars:
          VLLM_WORKER_MULTIPROC_METHOD: spawn

    - name: e5_language_classifier
      args:
        deployment_type: classifier_deployment
        model_definition:
          model_id: e5_language_classifier
          model_loader:
            s3_artifact:
              s3_key_prefix: model_artifacts/e5-language-classifier

    - name: llama31_70b_instruct_awq
      args:
        deployment_configs:
          LLMDeployment:
            env_options_override:
              dev:
                autoscaling_config:
                  min_replicas: 2
            gpu_type_options_override:
              A10G:
                ray_actor_options:
                  num_gpus: 4
              L40S:
                ray_actor_options:
                  num_gpus: 2
              T4:
                ray_actor_options:
                  num_gpus: 8
            options:
              autoscaling_config:
                min_replicas: 1
                target_ongoing_requests: 3
              max_ongoing_requests: 4
        deployment_type: vllm_text_gen_model_deployment
        model_definition:
          gpu_type_model_config_override:
            A10G:
              engine_args:
                gpu_memory_utilization: 0.95
                max_model_len: 28930
                tensor_parallel_size: 4
            L40S:
              engine_args:
                gpu_memory_utilization: 0.95
                max_model_len: 28930
                tensor_parallel_size: 2
            T4:
              engine_args:
                dtype: half
                max_model_len: 28930
                tensor_parallel_size: 8
          model_id: llama31_70b_instruct_awq
          model_loader:
            s3_artifact:
              s3_key_prefix: model_artifacts/llama31-70b-instruct-awq
        tokenizer_definition:
          model_id: llama31_70b_instruct_awq
          model_loader:
            s3_artifact:
              artifacts_list:
              - tokenizer_config.json
              - tokenizer.json
              s3_key_prefix: model_artifacts/llama31-70b-instruct-awq
      runtime_env:
        env_vars:
          VLLM_WORKER_MULTIPROC_METHOD: spawn

    - name: prompt_injection_cross_encoder
      args:
        deployment_configs:
          EncoderDeployment:
            options:
              ray_actor_options:
                num_gpus: 0.01
        deployment_type: encoder_deployment
        model_definition:
          model_id: prompt_injection_cross_encoder
          model_loader:
            s3_artifact:
              s3_key_prefix: model_artifacts/prompt-injection-cross-encoder-1114
          model_type: sentence_transformer_cross_encoder

    - name: prompt_injection_classifier
      args:
        deployment_type: classifier_deployment
        model_definition:
          custom_model_import_path: prompt_injection_classifier:PromptInjectionClassifier
          model_id: prompt_injection_classifier
          model_loader:
            s3_artifact:
              s3_key_prefix: model_artifacts/prompt-injection-classifier-01052025
          model_type: custom_model

    - name: bert_logparser_winlogbeat
      args:
        deployment_type: encoder_deployment
        model_definition:
          custom_model_import_path: bert_logparser_winlogbeat:BertLogparserWinlogbeatEncoder
          model_id: bert_logparser_winlogbeat
          model_loader:
            s3_artifact:
              s3_key_prefix: model_artifacts/bert-logparser-winlogbeat
          model_type: custom_model

    - name: llama33_70b_instruct_awq
      args:
        deployment_configs:
          LLMDeployment:
            gpu_type_options_override:
              A10G:
                ray_actor_options:
                  num_gpus: 4
              L40S:
                ray_actor_options:
                  num_gpus: 2
            options:
              autoscaling_config:
                min_replicas: 1
                target_ongoing_requests: 3
              max_ongoing_requests: 4
        deployment_type: vllm_text_gen_model_deployment
        model_definition:
          gpu_type_model_config_override:
            A10G:
              engine_args:
                gpu_memory_utilization: 0.95
                tensor_parallel_size: 4
              openai_serving_config:
                enable_auto_tools: true
                tool_parser: llama3_json
            L40S:
              engine_args:
                gpu_memory_utilization: 0.95
                tensor_parallel_size: 2
              openai_serving_config:
                enable_auto_tools: true
                tool_parser: llama3_json
          model_id: llama33_70b_instruct_awq
          model_loader:
            hugging_face_id:
              hf_model_id: casperhansen/llama-3.3-70b-instruct-awq
      runtime_env:
        env_vars:
          VLLM_WORKER_MULTIPROC_METHOD: spawn

    - name: qwq_32b
      args:
        deployment_configs:
          LLMDeployment:
            options:
              autoscaling_config:
                target_ongoing_requests: 1
              max_ongoing_requests: 2
              ray_actor_options:
                num_gpus: 4
          VLLMTextGenModelDeployment:
            options:
              ray_actor_options:
                num_cpus: 0.25
        deployment_type: vllm_text_gen_model_deployment
        model_definition:
          model_config:
            engine_args:
              tensor_parallel_size: 4
          model_id: qwq_32b
          model_loader:
            hugging_face_id:
              hf_model_id: Qwen/QwQ-32B
      runtime_env:
        env_vars:
          VLLM_WORKER_MULTIPROC_METHOD: spawn

    - name: openai_proxy
      import_path: main:SERVE_APP
      runtime_env:
        env_vars:
          ENABLE_AUTHZ: 'true'
          OPENAI_URL: '{{ .Values.openai.baseUrl }}'
          SECRETS_FILE_PATH: /home/ray/secrets.json`

// --- 5️⃣ ReconcileApplicationsConfigMap: bootstrap user‐editable apps fragment ---
func (b *Builder) ReconcileApplicationsConfigMap(ctx context.Context, p *aiApi.AIPlatform) error {
	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
		Name:      p.Name + "-applications",
		Namespace: p.Namespace,
	}}

	_, err := controllerutil.CreateOrUpdate(ctx, b.Client, cm, func() error {
		if cm.Data == nil {
			cm.Data = map[string]string{}
		}
		if _, exists := cm.Data["applications.yaml"]; !exists {
			application_file := os.Getenv("APPLICATION_FILE")
			content, err := os.ReadFile(application_file)
			if err != nil {
				return err
			}

			// Parse applications.yaml
			var apps RayServiceSpec
			if err := yaml.Unmarshal(content, &apps); err != nil {
				return fmt.Errorf("failed to parse applications.yaml: %w", err)
			}

			// Patch with default values
			PatchModelDefaults(&apps)

			// Marshal back to YAML
			patchedYaml, err := yaml.Marshal(&apps)
			if err != nil {
				return fmt.Errorf("failed to marshal patched applications.yaml: %w", err)
			}

			cm.Data["applications.yaml"] = string(patchedYaml)
		}

		return controllerutil.SetOwnerReference(p, cm, b.Scheme)
	})
	return err
}

func PatchModelDefaults(apps *RayServiceSpec) {
	for i := range apps.RayService.Applications {
		app := &apps.RayService.Applications[i]
		if app.Args == nil || app.Args.DeploymentConfigs == nil {
			continue // skip if no args or deployment configs
		}
		for deployName, deployConfig := range app.Args.DeploymentConfigs {
			for gpuType, cfg := range deployConfig.GPUTypeOptionsOverride {
				if cfg.RayActorOptions.NumGPUs == 0 {
					cfg.RayActorOptions.NumGPUs = 0.0 // default value
				}
				if cfg.RayActorOptions.NumCPUs == 0 {
					cfg.RayActorOptions.NumCPUs = 0.0 // default value
				}
				if cfg.AutoscalingConfig == nil {
					cfg.AutoscalingConfig = &AutoscalingConfig{}
				}
				if cfg.AutoscalingConfig.MinReplicas == nil {
					cfg.AutoscalingConfig.MinReplicas = new(int)
					*cfg.AutoscalingConfig.MinReplicas = 0
				}
				if cfg.AutoscalingConfig.MaxReplicas == nil {
					cfg.AutoscalingConfig.MaxReplicas = new(int)
					*cfg.AutoscalingConfig.MaxReplicas = 0
				}
				app.Args.DeploymentConfigs[deployName].GPUTypeOptionsOverride[gpuType] = cfg
			}
		}
	}
}

func BuildModelSpecsFromApplicationsYAML(data string) ([]ModelSpec, error) {
	var spec RayServiceSpec
	if err := yaml.Unmarshal([]byte(data), &spec); err != nil {
		return nil, err
	}

	var specs []ModelSpec
	for _, app := range spec.RayService.Applications {
		if app.Args == nil {
			continue // skip if no args
		}
		for _, config := range app.Args.DeploymentConfigs {
			if config.Options == nil || config.Options.RayActorOptions == nil {
				continue
			}
			numGPUs := config.Options.RayActorOptions.NumGPUs
			numCPUs := config.Options.RayActorOptions.NumCPUs

			// Skip only if both CPU and GPU are zero
			if numGPUs == 0 && numCPUs == 0 {
				continue
			}

			if config.Options.AutoscalingConfig == nil {
				config.Options.AutoscalingConfig = &AutoscalingConfig{}
			}
			replicas := config.Options.AutoscalingConfig.MinReplicas
			if replicas == nil {
				replicas = new(int)
				*replicas = 0 // default
			}

			tp := 1.0
			if numGPUs > 0.0 {
				tp = numGPUs
			}
			if config.Options.RayActorOptions == nil {
				config.Options.RayActorOptions = &RayActorOptions{}
			}
			memory := config.Options.RayActorOptions.Memory
			if memory == "" {
				memory = "8Gi" // default memory if not specified TODO FIXME: make this configurable
			}

			modelSpec := ModelSpec{
				Name:              app.Name,
				GPUsPerReplica:    numGPUs,
				TensorParallelism: tp,
				Replicas:          *replicas,
				CPU:               numCPUs,
				Memory:            memory,
			}

			if numCPUs > 0 {
				modelSpec.CPU = numCPUs
			}

			specs = append(specs, modelSpec)
		}
	}
	return specs, nil
}

func ReadApplicationsYAMLFromConfigMap(ctx context.Context, cl client.Client, name, namespace string) (string, error) {
	cm := &corev1.ConfigMap{}
	err := cl.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, cm)
	if err != nil {
		return "", err
	}
	val, ok := cm.Data["applications.yaml"]
	if !ok {
		return "", fmt.Errorf("applications.yaml not found in ConfigMap %s/%s", namespace, name)
	}
	return val, nil
}
