package schema

import (
	"github.com/rancher/norman/types"
	m "github.com/rancher/norman/types/mapper"
	"github.com/rancher/types/apis/cluster.cattle.io/v3/schema"
	"github.com/rancher/types/apis/management.cattle.io/v3"
	"github.com/rancher/types/factory"
	"github.com/rancher/types/mapper"
)

var (
	Version = types.APIVersion{
		Version: "v3",
		Group:   "management.cattle.io",
		Path:    "/v3",
		SubContexts: map[string]bool{
			"clusters": true,
		},
	}

	Schemas = factory.Schemas(&Version).
		Init(nodeTypes).
		Init(machineTypes).
		Init(authzTypes).
		Init(clusterTypes).
		Init(catalogTypes).
		Init(authnTypes).
		Init(schemaTypes).
		Init(stackTypes).
		Init(userTypes).
		Init(logTypes).
		Init(alertTypes).
		Init(globalTypes)
)

func schemaTypes(schemas *types.Schemas) *types.Schemas {
	return schemas.
		MustImport(&Version, v3.DynamicSchema{})
}

func catalogTypes(schemas *types.Schemas) *types.Schemas {
	return schemas.
		MustImport(&Version, v3.Catalog{}).
		MustImport(&Version, v3.Template{}).
		MustImport(&Version, v3.TemplateVersion{})
}

func nodeTypes(schemas *types.Schemas) *types.Schemas {
	return schema.NodeTypes(&Version, schemas)
}

func clusterTypes(schemas *types.Schemas) *types.Schemas {
	return schemas.
		AddMapperForType(&Version, v3.Cluster{},
			&m.Embed{Field: "status"},
			m.DisplayName{},
		).
		AddMapperForType(&Version, v3.ClusterStatus{},
			m.Drop{Field: "serviceAccountToken"},
			m.Drop{Field: "appliedSpec"},
			m.Drop{Field: "clusterName"},
		).
		AddMapperForType(&Version, v3.ClusterEvent{}, &m.Move{
			From: "type",
			To:   "eventType",
		}).
		MustImportAndCustomize(&Version, v3.Cluster{}, func(schema *types.Schema) {
			schema.SubContext = "clusters"
		}).
		MustImport(&Version, v3.ClusterEvent{}).
		MustImport(&Version, v3.ClusterRegistrationToken{}).
		MustImportAndCustomize(&Version, v3.Cluster{}, func(schema *types.Schema) {
			schema.MustCustomizeField("name", func(field types.Field) types.Field {
				field.Type = "dnsLabel"
				field.Nullable = true
				field.Required = false
				return field
			})
		})
}

func authzTypes(schemas *types.Schemas) *types.Schemas {
	return schemas.
		MustImport(&Version, v3.ProjectStatus{}).
		AddMapperForType(&Version, v3.Project{}, m.DisplayName{},
			&m.Embed{Field: "status"}).
		AddMapperForType(&Version, v3.GlobalRole{}, m.DisplayName{}).
		AddMapperForType(&Version, v3.RoleTemplate{}, m.DisplayName{}).
		AddMapperForType(&Version, v3.ProjectRoleTemplateBinding{},
			&mapper.NamespaceIDMapper{},
		).
		MustImportAndCustomize(&Version, v3.Project{}, func(schema *types.Schema) {
			schema.SubContext = "projects"
		}).
		MustImport(&Version, v3.GlobalRole{}).
		MustImport(&Version, v3.GlobalRoleBinding{}).
		MustImport(&Version, v3.RoleTemplate{}).
		MustImport(&Version, v3.PodSecurityPolicyTemplate{}).
		MustImport(&Version, v3.ClusterRoleTemplateBinding{}).
		MustImport(&Version, v3.ProjectRoleTemplateBinding{}).
		MustImport(&Version, v3.GlobalRoleBinding{})
}

func machineTypes(schemas *types.Schemas) *types.Schemas {
	return schemas.
		AddMapperForType(&Version, v3.MachineSpec{}, &m.Embed{Field: "nodeSpec"}).
		AddMapperForType(&Version, v3.MachineStatus{},
			&m.Drop{Field: "token"},
			&m.Drop{Field: "rkeNode"},
			&m.Drop{Field: "machineTemplateSpec"},
			&m.Drop{Field: "machineDriverConfig"},
			&m.Embed{Field: "nodeStatus"},
			&m.SliceMerge{From: []string{"conditions", "nodeConditions"}, To: "conditions"}).
		AddMapperForType(&Version, v3.MachineConfig{},
			&m.Drop{Field: "clusterName"}).
		AddMapperForType(&Version, v3.Machine{},
			&m.Embed{Field: "status"},
			m.DisplayName{}).
		AddMapperForType(&Version, v3.MachineDriver{}, m.DisplayName{}).
		AddMapperForType(&Version, v3.MachineTemplate{}, m.DisplayName{}).
		MustImport(&Version, v3.Machine{}).
		MustImportAndCustomize(&Version, v3.MachineDriver{}, func(schema *types.Schema) {
			schema.ResourceActions["activate"] = types.Action{
				Output: "machineDriver",
			}
			schema.ResourceActions["deactivate"] = types.Action{
				Output: "machineDriver",
			}
		}).
		MustImport(&Version, v3.MachineTemplate{})

}

func authnTypes(schemas *types.Schemas) *types.Schemas {
	return schemas.
		AddMapperForType(&Version, v3.User{}, m.DisplayName{}).
		AddMapperForType(&Version, v3.Group{}, m.DisplayName{}).
		MustImport(&Version, v3.Group{}).
		MustImport(&Version, v3.GroupMember{}).
		MustImport(&Version, v3.Principal{}).
		MustImport(&Version, v3.LoginInput{}).
		MustImport(&Version, v3.LocalCredential{}).
		MustImport(&Version, v3.GithubCredential{}).
		MustImport(&Version, v3.ChangePasswordInput{}).
		MustImport(&Version, v3.SetPasswordInput{}).
		MustImportAndCustomize(&Version, v3.Token{}, func(schema *types.Schema) {
			schema.CollectionActions = map[string]types.Action{
				"login": {
					Input:  "loginInput",
					Output: "token",
				},
				"logout": {},
			}
		}).
		MustImportAndCustomize(&Version, v3.User{}, func(schema *types.Schema) {
			schema.ResourceActions = map[string]types.Action{
				"setpassword": {
					Input:  "setPasswordInput",
					Output: "user",
				},
			}
			schema.CollectionActions = map[string]types.Action{
				"changepassword": {
					Input: "changePasswordInput",
				},
			}
		})
}

func stackTypes(schema *types.Schemas) *types.Schemas {
	return schema.
		MustImportAndCustomize(&Version, v3.Stack{}, func(schema *types.Schema) {
			schema.ResourceActions = map[string]types.Action{
				"upgrade": {
					Input: "templateVersionId",
				},
				"rollback": {
					Input: "revision",
				},
			}
		})
}

func userTypes(schema *types.Schemas) *types.Schemas {
	return schema.
		MustImportAndCustomize(&Version, v3.Preference{}, func(schema *types.Schema) {
			schema.MustCustomizeField("name", func(f types.Field) types.Field {
				f.Required = true
				return f
			})
			schema.MustCustomizeField("namespaceId", func(f types.Field) types.Field {
				f.Required = false
				return f
			})
		})
}

func logTypes(schema *types.Schemas) *types.Schemas {
	return schema.
		AddMapperForType(&Version, &v3.ClusterLogging{},
			m.DisplayName{}).
		AddMapperForType(&Version, &v3.ProjectLogging{},
			m.DisplayName{}).
		MustImport(&Version, v3.ClusterLogging{}).
		MustImport(&Version, v3.ProjectLogging{})
}

func globalTypes(schema *types.Schemas) *types.Schemas {
	return schema.
		AddMapperForType(&Version, v3.ListenConfig{},
			m.DisplayName{},
			m.Drop{Field: "caKey"},
			m.Drop{Field: "caCert"},
		).
		MustImport(&Version, v3.ListenConfig{}).
		MustImportAndCustomize(&Version, v3.Setting{}, func(schema *types.Schema) {
			schema.MustCustomizeField("name", func(f types.Field) types.Field {
				f.Required = true
				return f
			})
		})
}

func alertTypes(schema *types.Schemas) *types.Schemas {
	return schema.
		AddMapperForType(&Version, &v3.Notifier{},
			m.DisplayName{}).
		AddMapperForType(&Version, &v3.ClusterAlert{},
			&m.Embed{Field: "status"},
			m.DisplayName{}).
		AddMapperForType(&Version, &v3.ProjectAlert{},
			&m.Embed{Field: "status"},
			m.DisplayName{}).
		MustImportAndCustomize(&Version, v3.Notifier{}, func(schema *types.Schema) {
			schema.CollectionActions = map[string]types.Action{
				//Add a message body as input
				"send": {},
			}
		}).
		MustImportAndCustomize(&Version, v3.ClusterAlert{}, func(schema *types.Schema) {

			schema.MustCustomizeField("severity", func(f types.Field) types.Field {
				f.Required = true
				f.Type = "enum"
				f.Options = []string{"info", "critical", "warning"}
				return f
			})
			schema.MustCustomizeField("initialWaitSeconds", func(f types.Field) types.Field {
				f.Required = true
				f.Nullable = false
				f.Default = 180
				return f
			})
			schema.MustCustomizeField("repeatIntervalSeconds", func(f types.Field) types.Field {
				f.Required = true
				f.Nullable = false
				f.Default = 3600
				return f
			})
			schema.MustCustomizeField("recipientList", func(f types.Field) types.Field {
				f.Required = true
				f.Nullable = false
				return f
			})

			schema.ResourceActions = map[string]types.Action{
				"activate":   {},
				"deactivate": {},
				"mute":       {},
				"unmute":     {},
			}
		}).
		MustImportAndCustomize(&Version, v3.ProjectAlert{}, func(schema *types.Schema) {

			schema.MustCustomizeField("severity", func(f types.Field) types.Field {
				f.Required = true
				f.Type = "enum"
				f.Options = []string{"info", "critical", "warning"}
				return f
			})
			schema.MustCustomizeField("initialWaitSeconds", func(f types.Field) types.Field {
				f.Required = true
				f.Nullable = false
				f.Default = 180
				return f
			})
			schema.MustCustomizeField("repeatIntervalSeconds", func(f types.Field) types.Field {
				f.Required = true
				f.Nullable = false
				f.Default = 3600
				return f
			})
			schema.MustCustomizeField("recipientList", func(f types.Field) types.Field {
				f.Required = true
				f.Nullable = false
				return f
			})

			schema.ResourceActions = map[string]types.Action{
				"activate":   {},
				"deactivate": {},
				"mute":       {},
				"unmute":     {},
			}
		}).
		MustImportAndCustomize(&Version, v3.SmtpConfig{}, func(schema *types.Schema) {
			schema.MustCustomizeField("host", func(field types.Field) types.Field {
				field.Type = "dnsLabel"
				field.Nullable = false
				field.Required = true
				return field
			})
			schema.MustCustomizeField("port", func(field types.Field) types.Field {
				field.Nullable = false
				field.Required = true
				min := int64(1)
				max := int64(65535)
				field.Min = &min
				field.Max = &max
				return field
			})
			schema.MustCustomizeField("username", func(field types.Field) types.Field {
				field.Nullable = false
				field.Required = true
				return field
			})
			schema.MustCustomizeField("password", func(field types.Field) types.Field {
				field.Nullable = false
				field.Required = true
				field.Type = "masked"
				return field
			})
			schema.MustCustomizeField("tls", func(field types.Field) types.Field {
				field.Nullable = false
				field.Required = true
				return field
			})
		}).
		MustImportAndCustomize(&Version, v3.SlackConfig{}, func(schema *types.Schema) {
			schema.MustCustomizeField("url", func(field types.Field) types.Field {
				field.Nullable = false
				field.Required = true
				return field
			})
		}).
		MustImportAndCustomize(&Version, v3.WebhookConfig{}, func(schema *types.Schema) {
			schema.MustCustomizeField("url", func(field types.Field) types.Field {
				field.Nullable = false
				field.Required = true
				return field
			})
		}).
		MustImportAndCustomize(&Version, v3.PagerdutyConfig{}, func(schema *types.Schema) {
			schema.MustCustomizeField("serviceKey", func(field types.Field) types.Field {
				field.Nullable = false
				field.Required = true
				field.Type = "masked"
				return field
			})
		}).
		MustImportAndCustomize(&Version, v3.TargetSystemService{}, func(schema *types.Schema) {
			schema.MustCustomizeField("type", func(field types.Field) types.Field {
				field.Type = "enum"
				field.Options = []string{"dns", "etcd", "controller manager", "scheduler", "network"}
				field.Nullable = false
				field.Required = true
				return field
			})
		}).
		MustImportAndCustomize(&Version, v3.TargetWorkload{}, func(schema *types.Schema) {
			schema.MustCustomizeField("unavailablePercentage", func(field types.Field) types.Field {
				field.Nullable = false
				field.Required = true
				min := int64(1)
				max := int64(100)
				field.Min = &min
				field.Max = &max
				return field
			})
		}).
		MustImportAndCustomize(&Version, v3.TargetPod{}, func(schema *types.Schema) {
			schema.MustCustomizeField("id", func(field types.Field) types.Field {
				field.Nullable = false
				field.Required = true
				return field
			})
		})

}
