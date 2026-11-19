//go:build postgres

package frameworkcatalog

import (
	"context"
	"fmt"
)

// ListReferenceTemplates returns the bounded built-in guidance library. It is
// intentionally separate from the immutable, independently reviewed catalog.
func (r *PostgresRepository) ListReferenceTemplates(ctx context.Context, jurisdiction string) ([]ReferenceTemplate, error) {
	if r == nil || r.pool == nil || (jurisdiction != "ALL" && !publicationCodePattern.MatchString(jurisdiction)) {
		return nil, ErrInvalidPublication
	}
	rows, err := r.pool.Query(ctx, `SELECT t.template_code,t.title,t.description,t.publisher,t.edition,t.jurisdiction,t.topic,g.purpose,g.applicability_prompt,g.official_source,g.source_note,g.review_status
 FROM framework_reference_templates t JOIN framework_reference_guidance g USING(template_code)
 WHERE ($1 = 'ALL' OR t.jurisdiction = $1) ORDER BY t.template_code LIMIT 100`, jurisdiction)
	if err != nil {
		return nil, fmt.Errorf("reference templates: %w", err)
	}
	defer rows.Close()
	items := make([]ReferenceTemplate, 0)
	for rows.Next() {
		var t ReferenceTemplate
		if err = rows.Scan(&t.Code, &t.Title, &t.Description, &t.Publisher, &t.Edition, &t.Jurisdiction, &t.Topic, &t.Purpose, &t.ApplicabilityPrompt, &t.OfficialSource, &t.SourceNote, &t.ReviewStatus); err != nil {
			return nil, err
		}
		t.Areas = make([]ReferenceArea, 0)
		items = append(items, t)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	// Templates are an intentionally bounded platform catalog; include no more
	// than 100 starter areas per template and never infer complete coverage.
	for i := range items {
		areaRows, e := r.pool.Query(ctx, `SELECT area_code,title,assessment_prompt,evidence_examples FROM framework_reference_guidance_areas WHERE template_code=$1 ORDER BY ordinal LIMIT 100`, items[i].Code)
		if e != nil {
			return nil, e
		}
		for areaRows.Next() {
			var a ReferenceArea
			if e = areaRows.Scan(&a.Code, &a.Title, &a.AssessmentPrompt, &a.EvidenceExamples); e != nil {
				areaRows.Close()
				return nil, e
			}
			items[i].Areas = append(items[i].Areas, a)
		}
		e = areaRows.Err()
		areaRows.Close()
		if e != nil {
			return nil, e
		}
	}
	return items, nil
}
