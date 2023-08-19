package ai

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/geometry"
)

type Blackboard struct {
	ReportedIncidents map[string]core.IncidentReport
}

func (b *Blackboard) AddKnowledge(person *core.Actor, report core.IncidentReport) {
	if report == core.EmptyReport {
		return
	}
	tickOfNewReport := report.Tick
	if existingReport, doesExist := b.ReportedIncidents[report.Hash()]; doesExist {
		report = existingReport
		if report.Type.IsAnEvent() { // this might be a new incident
			report.Tick = geometry.UIntMax(existingReport.Tick, tickOfNewReport)
			timeSpanInTicks := geometry.Abs(int(existingReport.Tick - tickOfNewReport))
			if timeSpanInTicks > ebiten.TPS()*4.0 {
				// assume a new incident at the same location
				if report.FinishedHandling {
					report.FinishedHandling = false
					report.RegisteredHandler = nil
					println(fmt.Sprintf("REPEATED OCCURENCE -> %s", report.String()))
				}
			}
		} else { // reported the same incident multiple times
			report.Tick = geometry.UIntMin(existingReport.Tick, tickOfNewReport)
		}
	} else {
		println(fmt.Sprintf("NEW REPORT (%s) -> %s", report.Type.AbstractType(), report.String()))
	}

	if !report.KnownBy.Contains(person) {
		report.KnownBy.Add(person)
		//println(fmt.Sprintf("KNOWLEDGE GROUP CHANGED -> %s", report.String()))
	}

	b.ReportedIncidents[report.Hash()] = report
}

func (b *Blackboard) GetNextIncidentForCleanup(filter func(report core.IncidentReport) bool) core.IncidentReport {
	for _, report := range b.ReportedIncidents {
		if !report.FinishedHandling || !report.Type.NeedsCleanup() || report.RegisteredCleaner != nil {
			continue
		}
		if filter(report) {
			return report
		}
	}
	return core.EmptyReport
}

func (b *Blackboard) GetNextIncidentForSnitching(person *core.Actor) core.IncidentReport {
	for _, report := range b.ReportedIncidents {
		if report.FinishedHandling || report.RegisteredHandler != nil || report.RegisteredSnitch != nil || report.IsKnownByGuards() || !report.KnownBy.Contains(person) {
			continue
		}
		return report
	}
	return core.EmptyReport
}

func (b *Blackboard) IncidentsNeedCleanup(filter func(report core.IncidentReport) bool) bool {
	for _, report := range b.ReportedIncidents {
		if !report.FinishedHandling || !report.Type.NeedsCleanup() || report.RegisteredCleaner != nil {
			continue
		}
		if filter(report) {
			return true
		}
	}
	return false
}

func (b *Blackboard) RemoveIncidentReport(report core.IncidentReport) {
	delete(b.ReportedIncidents, report.Hash())
}
func (b *Blackboard) GetIncidentsForInvestigation(person *core.Actor, currentTick uint64) []core.IncidentReport {
	reports := []core.IncidentReport{}
	for _, report := range b.ReportedIncidents {
		age := currentTick - report.Tick
		if report.FinishedHandling || report.RegisteredHandler != nil || !report.KnownBy.Contains(person) || age > (uint64(ebiten.TPS()*60.0)) {
			continue
		}
		reports = append(reports, report)
	}
	return reports
}

func (b *Blackboard) RemoveOldIncidents(currentTick uint64) {
	for hash, report := range b.ReportedIncidents {
		age := currentTick - report.Tick
		if age > (uint64(ebiten.TPS() * 60.0)) {
			delete(b.ReportedIncidents, hash)
		}
	}
}

func (b *Blackboard) LastUnhandledReport(keep func(report core.IncidentReport) bool) (core.IncidentReport, bool) {
	lastKnownLocationReport := core.IncidentReport{}
	lastReportedAt := uint64(0)
	for _, report := range b.ReportedIncidents {
		if !keep(report) || report.FinishedHandling {
			continue
		}
		if report.HasActiveHandler() {
			currentlyHandling := false
			if stateOfHandler, ok := report.RegisteredHandler.AI.GetState().(InvestigationMovement); ok {
				currentlyHandling = stateOfHandler.Incident.Hash() == report.Hash()
			}
			if !currentlyHandling {
				continue
			}
		}

		if report.Tick > lastReportedAt {
			lastKnownLocationReport = report
			lastReportedAt = report.Tick
		}
	}
	return lastKnownLocationReport, lastReportedAt > 0
}

func (b *Blackboard) Filter(keep func(report core.IncidentReport) bool) []core.IncidentReport {
	var reports []core.IncidentReport
	for _, report := range b.ReportedIncidents {
		if keep(report) {
			reports = append(reports, report)
		}
	}
	return reports
}

func (b *Blackboard) TransferKnowledge(one *core.Actor, two *core.Actor) {
	for _, report := range b.ReportedIncidents {
		oneKnowsThis := report.KnownBy.Contains(one)
		twoKnowsThis := report.KnownBy.Contains(two)
		if !oneKnowsThis && !twoKnowsThis {
			continue
		}

		if oneKnowsThis {
			report.KnownBy.Add(two)
		} else if twoKnowsThis {
			report.KnownBy.Add(one)
		}

		b.ReportedIncidents[report.Hash()] = report
	}
}
