package settings

import (
	"embed"
	"errors"
	"fmt"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/bearer/bearer/api"
	"github.com/bearer/bearer/internal/flag"
	"github.com/bearer/bearer/internal/util/ignore"
	ignoretypes "github.com/bearer/bearer/internal/util/ignore/types"
	"github.com/bearer/bearer/internal/util/output"
	"github.com/bearer/bearer/internal/util/rego"
	"github.com/bearer/bearer/internal/version_check"

	globaltypes "github.com/bearer/bearer/internal/types"
)

var (
	Timeout                          = 10 * time.Minute  // "The maximum time alloted to complete the scan
	TimeoutFileMinimum               = 5 * time.Second   // Minimum timeout assigned for scanning each file. This config superseeds timeout-second-per-bytes
	TimeoutFileMaximum               = 30 * time.Second  // Maximum timeout assigned for scanning each file. This config superseeds timeout-second-per-bytes
	TimeoutFileBytesPerSecond        = 1 * 1000          // 1 Kb/s minimum number of bytes per second allowed to scan a file
	TimeoutWorkerFileGrace           = 5 * time.Second   // Grace period to allow a worker to timeout on it's own
	TimeoutWorkerOnline              = 60 * time.Second  // Maximum time to wait for a worker process to come online
	TimeoutWorkerShutdown            = 5 * time.Second   // Maximum time to wait for a worker process to shut down cleanly
	CodeExtractBuffer                = 3                 // Number of lines allowed before or after the detection
	FileSizeMaximum                  = 2 * 1000 * 1000   // 2 MB Ignore files larger than the specified value
	FilesPerWorker                   = 1000              // By default, start a worker per this many files, up to the number of CPUs
	MemorySoftMaximum         uint64 = 650 * 1000 * 1000 // 650 MB If the memory needed to scan a file surpasses the specified limit, ask the worker to reduce memory usage.
	MemoryMaximum             uint64 = 800 * 1000 * 1000 // 800 MB If the memory needed to scan a file surpasses the specified limit, skip the file.
	ExistingWorker                   = ""                // Specify the URL of an existing worker
)

type WorkerOptions struct {
	Timeout                   time.Duration `mapstructure:"timeout" json:"timeout" yaml:"timeout"`
	TimeoutFileMinimum        time.Duration `mapstructure:"timeout-file-min" json:"timeout-file-min" yaml:"timeout-file-min"`
	TimeoutFileMaximum        time.Duration `mapstructure:"timeout-file-max"  json:"timeout-file-max" yaml:"timeout-file-max"`
	TimeoutFileBytesPerSecond int           `mapstructure:"timeout-file-bytes-per-second" json:"timeout-file-bytes-per-second" yaml:"timeout-file-bytes-per-second"`
	TimeoutWorkerOnline       time.Duration `mapstructure:"timeout-worker-online" json:"timeout-worker-online" yaml:"timeout-worker-online"`
	FileSizeMaximum           int           `mapstructure:"file-size-max" json:"file-size-max" yaml:"file-size-max"`
	ExistingWorker            string        `mapstructure:"existing-worker" json:"existing-worker" yaml:"existing-worker"`
}

type Config struct {
	Client                     *api.API
	Worker                     WorkerOptions                             `mapstructure:"worker" json:"worker" yaml:"worker"`
	Scan                       flag.ScanOptions                          `mapstructure:"scan" json:"scan" yaml:"scan"`
	Report                     flag.ReportOptions                        `mapstructure:"report" json:"report" yaml:"report"`
	IgnoredFingerprints        map[string]ignoretypes.IgnoredFingerprint `mapstructure:"ignored_fingerprints" json:"ignored_fingerprints" yaml:"ignored_fingerprints"`
	StaleIgnoredFingerprintIds []string                                  `mapstructure:"stale_ignored_fingerprint_ids" json:"stale_ignored_fingerprint_ids" yaml:"stale_ignored_fingerprint_ids"`
	CloudIgnoresUsed           bool                                      `mapstructure:"cloud_ignores_used" json:"cloud_ignores_used" yaml:"cloud_ignores_used"`
	Policies                   map[string]*Policy                        `mapstructure:"policies" json:"policies" yaml:"policies"`
	Target                     string                                    `mapstructure:"target" json:"target" yaml:"target"`
	IgnoreFile                 string                                    `mapstructure:"ignore_file" json:"ignore_file" yaml:"ignore_file"`
	Rules                      map[string]*Rule                          `mapstructure:"rules" json:"rules" yaml:"rules"`
	BuiltInRules               map[string]*Rule                          `mapstructure:"built_in_rules" json:"built_in_rules" yaml:"built_in_rules"`
	CacheUsed                  bool                                      `mapstructure:"cache_used" json:"cache_used" yaml:"cache_used"`
	BearerRulesVersion         string                                    `mapstructure:"bearer_rules_version" json:"bearer_rules_version" yaml:"bearer_rules_version"`
	NoColor                    bool                                      `mapstructure:"no_color" json:"no_color" yaml:"no_color"`
	Debug                      bool                                      `mapstructure:"debug" json:"debug" yaml:"debug"`
	LogLevel                   string                                    `mapstructure:"string" json:"string" yaml:"string"`
	DebugProfile               bool                                      `mapstructure:"debug_profile" json:"debug_profile" yaml:"debug_profile"`
}

type Modules []*PolicyModule

type Policy struct {
	Type    string  `mapstructure:"type" json:"type" yaml:"type"`
	Query   string  `mapstructure:"query" json:"query" yaml:"query"`
	Modules Modules `mapstructure:"modules" json:"modules" yaml:"modules"`
}

type PolicyModule struct {
	Path    string `mapstructure:"path" json:"path,omitempty" yaml:"path,omitempty"`
	Name    string `mapstructure:"name" json:"name" yaml:"name"`
	Content string `mapstructure:"content" json:"content" yaml:"content"`
}

type MatchOn string

const (
	PRESENCE          MatchOn = "presence"
	ABSENCE           MatchOn = "absence"
	STORED_DATA_TYPES MatchOn = "stored_data_types"
)

type RuleReferenceScope string

const (
	CURSOR_STRICT_SCOPE RuleReferenceScope = "cursor_strict"
	CURSOR_SCOPE        RuleReferenceScope = "cursor"
	NESTED_SCOPE        RuleReferenceScope = "nested"
	NESTED_STRICT_SCOPE RuleReferenceScope = "nested_strict"
	RESULT_SCOPE        RuleReferenceScope = "result"

	DefaultScope = NESTED_SCOPE
)

type LoadRulesResult struct {
	BuiltInRules       map[string]*Rule
	Rules              map[string]*Rule
	CacheUsed          bool
	BearerRulesVersion string
}

type RuleTrigger struct {
	MatchOn           MatchOn `mapstructure:"match_on" json:"match_on" yaml:"match_on"`
	DataTypesRequired bool    `mapstructure:"data_types_required" json:"data_types_required" yaml:"data_types_required"`
	RequiredDetection *string `mapstructure:"required_detection" json:"required_detection" yaml:"required_detection"`
}

type RuleDefinitionTrigger struct {
	MatchOn           *MatchOn `mapstructure:"match_on" json:"match_on" yaml:"match_on"`
	RequiredDetection *string  `mapstructure:"required_detection" json:"required_detection" yaml:"required_detection"`
	DataTypesRequired *bool    `mapstructure:"data_types_required" json:"data_types_required" yaml:"data_types_required"`
}

type RuleMetadata struct {
	Description        string   `mapstructure:"description" json:"description" yaml:"description"`
	RemediationMessage string   `mapstructure:"remediation_message" json:"remediation_message" yaml:"remediation_message"`
	CWEIDs             []string `mapstructure:"cwe_id" json:"cwe_id" yaml:"cwe_id"`
	AssociatedRecipe   string   `mapstructure:"associated_recipe" json:"associated_recipe" yaml:"associated_recipe"`
	ID                 string   `mapstructure:"id" json:"id" yaml:"id"`
	DocumentationUrl   string   `mapstructure:"documentation_url" json:"documentation_url" yaml:"documentation_url"`
}

type RuleDefinition struct {
	Disabled           bool                   `mapstructure:"disabled" json:"disabled" yaml:"disabled"`
	Type               string                 `mapstructure:"type" json:"type" yaml:"type"`
	Languages          []string               `mapstructure:"languages" json:"languages" yaml:"languages"`
	Imports            []string               `mapstructure:"imports" json:"imports" yaml:"imports"`
	ParamParenting     bool                   `mapstructure:"param_parenting" json:"param_parenting" yaml:"param_parenting"`
	Patterns           []RulePattern          `mapstructure:"patterns" json:"patterns" yaml:"patterns"`
	SanitizerRuleID    string                 `mapstructure:"sanitizer" json:"sanitizer" yaml:"sanitizer"`
	Stored             bool                   `mapstructure:"stored" json:"stored" yaml:"stored"`
	Detectors          []string               `mapstructure:"detectors" json:"detectors,omitempty" yaml:"detectors,omitempty"`
	Processors         []string               `mapstructure:"processors" json:"processors,omitempty" yaml:"processors,omitempty"`
	AutoEncrytPrefix   string                 `mapstructure:"auto_encrypt_prefix" json:"auto_encrypt_prefix,omitempty" yaml:"auto_encrypt_prefix,omitempty"`
	DetectPresence     bool                   `mapstructure:"detect_presence" json:"detect_presence" yaml:"detect_presence"`
	Trigger            *RuleDefinitionTrigger `mapstructure:"trigger" json:"trigger" yaml:"trigger"` // TODO: use enum value
	Severity           string                 `mapstructure:"severity" json:"severity,omitempty" yaml:"severity,omitempty"`
	SkipDataTypes      []string               `mapstructure:"skip_data_types" json:"skip_data_types,omitempty" yaml:"skip_data_types,omitempty"`
	OnlyDataTypes      []string               `mapstructure:"only_data_types" json:"only_data_types,omitempty" yaml:"only_data_types,omitempty"`
	HasDetailedContext bool                   `mapstructure:"has_detailed_context" json:"has_detailed_context,omitempty" yaml:"has_detailed_context,omitempty"`
	Metadata           *RuleMetadata          `mapstructure:"metadata" json:"metadata" yaml:"metadata"`
	Auxiliary          []Auxiliary            `mapstructure:"auxiliary" json:"auxiliary" yaml:"auxiliary"`
	DependencyCheck    bool                   `mapstructure:"dependency_check" json:"dependency_check" yaml:"dependency_check"`
	Dependency         *Dependency            `mapstructure:"dependency" json:"dependency" yaml:"dependency"`
}

type Dependency struct {
	Filename   string `mapstructure:"filename" json:"filename" yaml:"filename"`
	Name       string `mapstructure:"name" json:"name" yaml:"name"`
	MinVersion string `mapstructure:"min_version" json:"min_version" yaml:"min_version"`
}

type Auxiliary struct {
	Id              string        `mapstructure:"id" json:"id" yaml:"id"`
	Type            string        `mapstructure:"type" json:"type" yaml:"type"`
	Languages       []string      `mapstructure:"languages" json:"languages" yaml:"languages"`
	Patterns        []RulePattern `mapstructure:"patterns" json:"patterns" yaml:"patterns"`
	SanitizerRuleID string        `mapstructure:"sanitizer" json:"sanitizer" yaml:"sanitizer"`

	RootSingularize bool `mapstructure:"root_singularize" yaml:"root_singularize" `
	RootLowercase   bool `mapstructure:"root_lowercase" yaml:"root_lowercase"`

	Stored           bool     `mapstructure:"stored" json:"stored" yaml:"stored"`
	Detectors        []string `mapstructure:"detectors" json:"detectors,omitempty" yaml:"detectors,omitempty"`
	Processors       []string `mapstructure:"processors" json:"processors,omitempty" yaml:"processors,omitempty"`
	AutoEncrytPrefix string   `mapstructure:"auto_encrypt_prefix" json:"auto_encrypt_prefix,omitempty" yaml:"auto_encrypt_prefix,omitempty"`

	// FIXME: remove after refactor of sql
	ParamParenting bool `mapstructure:"param_parenting" json:"param_parenting" yaml:"param_parenting"`
	DetectPresence bool `mapstructure:"detect_presence" json:"detect_presence" yaml:"detect_presence"`
	OmitParent     bool `mapstructure:"omit_parent" json:"omit_parent,omitempty" yaml:"omit_parent,omitempty"`
}

type Rule struct {
	Id                 string        `mapstructure:"id" json:"id,omitempty" yaml:"id,omitempty"`
	AssociatedRecipe   string        `mapstructure:"associated_recipe" json:"associated_recipe" yaml:"associated_recipe"`
	Type               string        `mapstructure:"type" json:"type,omitempty" yaml:"type,omitempty"` // TODO: use enum value
	Trigger            RuleTrigger   `mapstructure:"trigger" json:"trigger,omitempty" yaml:"trigger,omitempty"`
	IsLocal            bool          `mapstructure:"is_local" json:"is_local,omitempty" yaml:"is_local,omitempty"`
	Detectors          []string      `mapstructure:"detectors" json:"detectors,omitempty" yaml:"detectors,omitempty"`
	Processors         []string      `mapstructure:"processors" json:"processors,omitempty" yaml:"processors,omitempty"`
	Stored             bool          `mapstructure:"stored" json:"stored,omitempty" yaml:"stored,omitempty"`
	AutoEncrytPrefix   string        `mapstructure:"auto_encrypt_prefix" json:"auto_encrypt_prefix,omitempty" yaml:"auto_encrypt_prefix,omitempty"`
	HasDetailedContext bool          `mapstructure:"has_detailed_context" json:"has_detailed_context,omitempty" yaml:"has_detailed_context,omitempty"`
	SkipDataTypes      []string      `mapstructure:"skip_data_types" json:"skip_data_types,omitempty" yaml:"skip_data_types,omitempty"`
	OnlyDataTypes      []string      `mapstructure:"only_data_types" json:"only_data_types,omitempty" yaml:"only_data_types,omitempty"`
	Severity           string        `mapstructure:"severity" json:"severity,omitempty" yaml:"severity,omitempty"`
	Description        string        `mapstructure:"description" json:"description" yaml:"description"`
	RemediationMessage string        `mapstructure:"remediation_message" json:"remediation_messafe" yaml:"remediation_messafe"`
	CWEIDs             []string      `mapstructure:"cwe_ids" json:"cwe_ids" yaml:"cwe_ids"`
	Languages          []string      `mapstructure:"languages" json:"languages" yaml:"languages"`
	Patterns           []RulePattern `mapstructure:"patterns" json:"patterns" yaml:"patterns"`
	SanitizerRuleID    string        `mapstructure:"sanitizer" json:"sanitizer" yaml:"sanitizer"`
	DocumentationUrl   string        `mapstructure:"documentation_url" json:"documentation_url" yaml:"documentation_url"`
	IsAuxilary         bool          `mapstructure:"is_auxilary" json:"is_auxilary" yaml:"is_auxilary"`
	DependencyCheck    bool          `mapstructure:"dependency_check" json:"dependency_check" yaml:"dependency_check"`
	Dependency         *Dependency   `mapstructure:"dependency" json:"dependency" yaml:"dependency"`

	// FIXME: remove after refactor of sql
	Metavars       map[string]MetaVar `mapstructure:"metavars" json:"metavars" yaml:"metavars"`
	ParamParenting bool               `mapstructure:"param_parenting" json:"param_parenting" yaml:"param_parenting"`
	DetectPresence bool               `mapstructure:"detect_presence" json:"detect_presence" yaml:"detect_presence"`
	OmitParent     bool               `mapstructure:"omit_parent" json:"omit_parent" yaml:"omit_parent"`
}

type RuleReferenceImport struct {
	Variable string `mapstructure:"variable" json:"variable" yaml:"variable"`
	As       string `mapstructure:"as" json:"as" yaml:"as"`
}

type PatternFilter struct {
	Not       *PatternFilter        `mapstructure:"not" json:"not" yaml:"not"`
	Either    []PatternFilter       `mapstructure:"either" json:"either" yaml:"either"`
	Variable  string                `mapstructure:"variable" json:"variable" yaml:"variable"`
	Detection string                `mapstructure:"detection" json:"detection" yaml:"detection"`
	Scope     RuleReferenceScope    `mapstructure:"scope" json:"scope" yaml:"scope"`
	Filters   []PatternFilter       `mapstructure:"filters" json:"filters" yaml:"filters"`
	Imports   []RuleReferenceImport `mapstructure:"imports" json:"imports" yaml:"imports"`
	// Contains is deprecated in favour of Scope
	Contains           *bool    `mapstructure:"contains" json:"contains" yaml:"contains"`
	Regex              *Regexp  `mapstructure:"regex" json:"regex" yaml:"regex"`
	Values             []string `mapstructure:"values" json:"values" yaml:"values"`
	LengthLessThan     *int     `mapstructure:"length_less_than" json:"length_less_than" yaml:"length_less_than"`
	LessThan           *int     `mapstructure:"less_than" json:"less_than" yaml:"less_than"`
	LessThanOrEqual    *int     `mapstructure:"less_than_or_equal" json:"less_than_or_equal" yaml:"less_than_or_equal"`
	GreaterThan        *int     `mapstructure:"greater_than" json:"greater_than" yaml:"greater_than"`
	GreaterThanOrEqual *int     `mapstructure:"greater_than_or_equal" json:"greater_than_or_equal" yaml:"greater_than_or_equal"`
	StringRegex        *Regexp  `mapstructure:"string_regex" json:"string_regex" yaml:"string_regex"`
	FilenameRegex      *Regexp  `mapstructure:"filename_regex" json:"filename_regex" yaml:"filename_regex"`
}

type RulePattern struct {
	Pattern string          `mapstructure:"pattern" json:"pattern" yaml:"pattern"`
	Focus   string          `mapstructure:"focus" json:"focus" yaml:"focus"`
	Filters []PatternFilter `mapstructure:"filters" json:"filters" yaml:"filters"`
}

type Processor struct {
	Query   string  `mapstructure:"query" json:"query" yaml:"query"`
	Modules Modules `mapstructure:"modules" json:"modules" yaml:"modules"`
}

type MetaVar struct {
	Input  string `mapstructure:"input" json:"input" yaml:"input"`
	Output int    `mapstructure:"output" json:"output" yaml:"output"`
	Regex  string `mapstructure:"regex" json:"regex" yaml:"regex"`
}

//go:embed policies.yml
var defaultPolicies []byte

//go:embed built_in_rules/*
var buildInRulesFs embed.FS

//go:embed policies/*
var policiesFs embed.FS

//go:embed processors/*
var processorsFs embed.FS

func (rule *Rule) PolicyType() bool {
	return rule.Type == "risk"
}

func (rule *Rule) GetSeverity() string {
	if rule.Severity == "" {
		return globaltypes.LevelLow
	}

	return rule.Severity
}

func (rule *Rule) Language() string {
	if rule.Languages == nil {
		return "secret"
	}

	switch rule.Languages[0] {
	case "java":
		return "Java"
	case "javascript":
		return "JavaScript"
	case "ruby":
		return "Ruby"
	case "sql":
		return "SQL"
	default:
		return rule.Languages[0]
	}
}

func defaultWorkerOptions() WorkerOptions {
	return WorkerOptions{
		Timeout:                   Timeout,
		TimeoutFileMinimum:        TimeoutFileMinimum,
		TimeoutFileMaximum:        TimeoutFileMaximum,
		TimeoutFileBytesPerSecond: TimeoutFileBytesPerSecond,
		TimeoutWorkerOnline:       TimeoutWorkerOnline,
		FileSizeMaximum:           FileSizeMaximum,
		ExistingWorker:            ExistingWorker,
	}
}

func FromOptions(opts flag.Options, versionMeta *version_check.VersionMeta) (Config, error) {
	policies := DefaultPolicies()
	workerOptions := defaultWorkerOptions()
	result, err := loadRules(
		opts.ExternalRuleDir,
		opts.RuleOptions,
		versionMeta,
		opts.ScanOptions.Force,
	)
	if err != nil {
		return Config{}, err
	}

	for key := range policies {
		policy := policies[key]

		for _, module := range policy.Modules {
			if module.Path != "" {
				content, err := policiesFs.ReadFile(module.Path)
				if err != nil {
					return Config{}, err
				}
				module.Content = string(content)
			}
		}
	}

	ignoredFingerprints, _, _, err := ignore.GetIgnoredFingerprints(opts.GeneralOptions.IgnoreFile, &opts.ScanOptions.Target)
	if err != nil {
		return Config{}, err
	}

	config := Config{
		Client:              opts.Client,
		Worker:              workerOptions,
		Scan:                opts.ScanOptions,
		Report:              opts.ReportOptions,
		IgnoredFingerprints: ignoredFingerprints,
		NoColor:             opts.GeneralOptions.NoColor || opts.ReportOptions.Output != "",
		DebugProfile:        opts.GeneralOptions.DebugProfile,
		Debug:               opts.GeneralOptions.Debug,
		LogLevel:            opts.GeneralOptions.LogLevel,
		IgnoreFile:          opts.GeneralOptions.IgnoreFile,
		Policies:            policies,
		Rules:               result.Rules,
		BuiltInRules:        result.BuiltInRules,
		CacheUsed:           result.CacheUsed,
		BearerRulesVersion:  result.BearerRulesVersion,
	}

	if config.Scan.DiffBaseBranch != "" {
		if config.Report.Report != flag.ReportSecurity {
			return Config{}, errors.New("diff base branch is only supported for the security report")
		}

		if config.Client != nil {
			return Config{}, errors.New("diff base branch is not supported when using an api key")
		}
	}

	return config, nil
}

func (rulePattern *RulePattern) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Try to parse as a string
	var pattern string
	if err := unmarshal(&pattern); err == nil {
		rulePattern.Pattern = pattern
		return nil
	}

	// Wasn't a string so it must be the structured format
	type rawRulePattern RulePattern
	return unmarshal((*rawRulePattern)(rulePattern))
}

func (filter *PatternFilter) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type wrapper PatternFilter
	var wrapped wrapper
	if err := unmarshal(&wrapped); err != nil {
		return err
	}

	*filter = PatternFilter(wrapped)

	// Default Scope to "contains" and maintain backwards compatibility with rules
	// using the `contains` flag
	if filter.Detection != "" {
		if filter.Contains != nil {
			if !*filter.Contains {
				filter.Scope = CURSOR_SCOPE
			}
		}
		if filter.Scope == "" {
			filter.Scope = NESTED_SCOPE
		}
	}

	return nil
}

func DefaultPolicies() map[string]*Policy {
	policies := make(map[string]*Policy)
	var policy []*Policy

	err := yaml.Unmarshal(defaultPolicies, &policy)
	if err != nil {
		output.Fatal(fmt.Sprintf("failed to unmarshal policy file %s", err))
	}

	for _, policy := range policy {
		policies[policy.Type] = policy
	}

	return policies
}

func ProcessorRegoModuleText(processorName string) (string, error) {
	processorPath := fmt.Sprintf("processors/%s.rego", processorName)
	data, err := processorsFs.ReadFile(processorPath)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (modules Modules) ToRegoModules() (output []rego.Module) {
	for _, module := range modules {
		output = append(output, rego.Module{
			Name:    module.Name,
			Content: module.Content,
		})
	}
	return
}