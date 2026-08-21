package monitoring

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"
)

type Report struct {
	Portfolios []PortfolioInfo
	Positions  []PoitionInfo
}

type PortfolioInfo struct {
	Client         string
	Portfolio      string
	Amount         float64
	VarMargin      float64
	VarMarginRatio float64
	UsedRatio      float64
}

type PoitionInfo struct {
	Client    string
	Portfolio string
	Security  string
	Deadline  time.Time
	Planned   int
	Actual    int
	Ok        bool
}

func formatReport(r Report) string {
	var sb = &strings.Builder{}

	var w = newTabWriter(sb)
	fmt.Fprintf(w, "Client\tPortfolio\tAmount\tVarMargin\tVarMarginRatio\tUsedRatio\t\n")
	for _, portfolio := range r.Portfolios {
		fmt.Fprintf(w, "%v\t%v\t%.0f\t%.0f\t%.1f\t%.1f\t\n",
			portfolio.Client,
			portfolio.Portfolio,
			portfolio.Amount,
			portfolio.VarMargin,
			portfolio.VarMarginRatio*100,
			portfolio.UsedRatio*100,
		)
	}
	w.Flush()
	fmt.Fprintln(sb, "Total portfolios:", len(r.Portfolios))

	w = newTabWriter(sb)
	fmt.Fprintf(w, "Client\tPortfolio\tSecurity\tPlanned\tActual\tStatus\t\n")
	for _, position := range r.Positions {
		var status string
		if position.Ok {
			status = "+"
		} else {
			status = "!"
		}
		fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\t\n",
			position.Client,
			position.Portfolio,
			position.Security,
			position.Planned,
			position.Actual,
			status,
		)
	}
	w.Flush()
	fmt.Fprintln(sb, "Total strategies:", len(r.Positions))

	return sb.String()
}

func newTabWriter(w io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(w, 0, 0, 1, ' ', tabwriter.AlignRight)
}
