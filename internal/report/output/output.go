package output

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hhatto/gocloc"

	"github.com/bearer/bearer/internal/commands/process/settings"
	"github.com/bearer/bearer/internal/flag"
	"github.com/bearer/bearer/internal/report/basebranchfindings"
	globaltypes "github.com/bearer/bearer/internal/types"

	"github.com/bearer/bearer/internal/report/output/dataflow"
	"github.com/bearer/bearer/internal/report/output/detectors"
	"github.com/bearer/bearer/internal/report/output/privacy"
	"github.com/bearer/bearer/internal/report/output/saas"
	"github.com/bearer/bearer/internal/report/output/security"
	"github.com/bearer/bearer/internal/report/output/stats"
	"github.com/bearer/bearer/internal/report/output/types"
)

var ErrUndefinedFormat = errors.New("undefined output format")

func GetData(
	report globaltypes.Report,
	config settings.Config,
	baseBranchFindings *basebranchfindings.Findings,
) (*types.ReportData, error) {
	data := &types.ReportData{}
	// add detectors
	err := detectors.AddReportData(data, report, config)
	if config.Report.Report == flag.ReportDetectors || err != nil {
		return data, err
	}

	// add dataflow to data
	if err = GetDataflow(data, report, config, config.Report.Report != flag.ReportDataFlow); err != nil {
		return data, err
	}

	// add report-specific items
	attemptCloudUpload := false
	switch config.Report.Report {
	case flag.ReportDataFlow:
		return data, err
	case flag.ReportSecurity:
		attemptCloudUpload = true
		err = security.AddReportData(data, config, baseBranchFindings)
	case flag.ReportSaaS:
		if err = security.AddReportData(data, config, baseBranchFindings); err != nil {
			return nil, err
		}

		attemptCloudUpload = true
		err = saas.GetReport(data, config, false)
	case flag.ReportPrivacy:
		err = privacy.AddReportData(data, config)
	case flag.ReportStats:
		err = stats.AddReportData(data, report.Inputgocloc, config)
	default:
		return nil, fmt.Errorf(`--report flag "%s" is not supported`, config.Report.Report)
	}

	if attemptCloudUpload && config.Client != nil && config.Client.Error == nil {
		// send SaaS report to Cloud
		data.SendToCloud = true
		saas.SendReport(config, data)
	}

	return data, err
}

func GetDataflow(reportData *types.ReportData, report globaltypes.Report, config settings.Config, isInternal bool) error {
	if reportData.Detectors == nil {
		if err := detectors.AddReportData(reportData, report, config); err != nil {
			return err
		}
	}
	for _, detection := range reportData.Detectors {
		detection.(map[string]interface{})["id"] = uuid.NewString()
	}
	return dataflow.AddReportData(reportData, config, isInternal)
}

func FormatOutput(
	reportData *types.ReportData,
	config settings.Config,
	cacheUsed bool,
	goclocResult *gocloc.Result,
	startTime time.Time,
	endTime time.Time,
) (string, error) {
	var formatter types.GenericFormatter
	switch config.Report.Report {
	case flag.ReportDetectors:
		formatter = detectors.NewFormatter(reportData, config)
	case flag.ReportDataFlow:
		formatter = dataflow.NewFormatter(reportData, config)
	case flag.ReportSecurity:
		formatter = security.NewFormatter(reportData, config, goclocResult, startTime, endTime)
	case flag.ReportPrivacy:
		formatter = privacy.NewFormatter(reportData, config)
	case flag.ReportSaaS:
		formatter = saas.NewFormatter(reportData, config)
	case flag.ReportStats:
		formatter = stats.NewFormatter(reportData, config)
	default:
		return "", fmt.Errorf(`--report flag "%s" is not supported`, config.Report.Report)
	}

	formatStr, err := formatter.Format(config.Report.Format)
	if err != nil {
		return formatStr, err
	}
	if formatStr == "" {
		return "", fmt.Errorf(`--report flag "%s" does not support --format flag "%s"`, config.Report.Report, config.Report.Format)
	}

	if !config.Scan.Quiet && (reportData.SendToCloud || cacheUsed) {
		// add cached data warning message
		if cacheUsed {
			formatStr += "\n\nCached data used (no code changes detected). Unexpected? Use --force to force a re-scan."
		}

		// add cloud info message
		if reportData.SendToCloud {
			if config.Client.Error == nil {
				formatStr += "\n\nData successfully sent to Bearer Cloud."
			} else {
				// client error
				formatStr += fmt.Sprintf("\n\nFailed to send data to Bearer Cloud. %s ", *config.Client.Error)
			}
		}

		formatStr += "\n"
	}

	return formatStr, err
}