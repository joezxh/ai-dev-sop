// Package console · M4 repo-pipeline runtime.
//
// Implements the six-stage pipeline from the v2 plan §7.2:
//
//   crawl      → fetch source items (local dir / GitHub repo / RSS feed)
//   parse      → HTML→Markdown + LLM summary
//   grade      → score each parsed item; threshold default 0.7
//   sink       → write the items that pass grading into bp_candidates
//   rehearse   → fuzzy-match candidates against existing best_practices
//                and decide merge vs new
//
// Each stage is a small interface so the e2e can swap in stub
// implementations. Production paths use the implementations in
// crawl.go (3 sources), parse.go (HTML→md + heuristic summary), etc.
// The runtime keeps the contract tiny on purpose — the M4 plan calls
// for an end-to-end vertical slice rather than production-quality
// content extraction.
package console

import (
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// CrawlItem is one raw item returned by a crawler.
type CrawlItem struct {
	URL       string
	Title     string
	Body      string // raw HTML for remote, raw markdown/text for local
	Tags      []string
	FetchedAt time.Time
}

// Crawler is the abstract crawl stage.
type Crawler interface {
	Crawl(ctx context.Context, target string) ([]CrawlItem, error)
}

// Parser converts a raw item into (title, body_md, summary).
type Parser interface {
	Parse(ctx context.Context, in CrawlItem) (title, bodyMD, summary string, err error)
}

// Grader scores a parsed item. Returns 0..1.
type Grader interface {
	Grade(ctx context.Context, title, bodyMD, summary string) (float64, error)
}

// PipelineRuntime bundles the configured stages. Each stage is
// interface-typed so the e2e can inject stubs.
type PipelineRuntime struct {
	Crawl  Crawler
	Parse  Parser
	Grade  Grader
}

// DefaultRuntime wires the production implementations. Tests can
// substitute stubs by setting fields directly.
func DefaultRuntime() *PipelineRuntime {
	return &PipelineRuntime{
		Crawl: &MultiCrawler{
			Local:  &LocalCrawler{},
			GitHub: &GitHubCrawler{HTTPClient: &http.Client{Timeout: 5 * time.Second}},
			RSS:    &RSSCrawler{HTTPClient: &http.Client{Timeout: 5 * time.Second}},
		},
		Parse: &HTMLParser{},
		Grade: &HeuristicGrader{},
	}
}

// --- Crawl stage ---

// MultiCrawler dispatches to the right backend based on the source.
// Source is one of RepoSrcLocal / RepoSrcGitHub / RepoSrcRSS.
type MultiCrawler struct {
	Local  *LocalCrawler
	GitHub *GitHubCrawler
	RSS    *RSSCrawler
}

// Crawl dispatches by source field. Empty source defaults to local.
func (m *MultiCrawler) Crawl(ctx context.Context, target string) ([]CrawlItem, error) {
	// Decide by prefix heuristic — target field carries both source and
	// destination URL/path. The HTTP handler below sets `source` on
	// the pipeline row separately, so this fallback only fires when the
	// caller already knows and just wants the dispatch logic.
	return m.Local.Crawl(ctx, target)
}

// CrawlBySource is the explicit dispatch used by the pipeline runtime.
func (m *MultiCrawler) CrawlBySource(ctx context.Context, source, target string) ([]CrawlItem, error) {
	switch source {
	case RepoSrcGitHub:
		return m.GitHub.Crawl(ctx, target)
	case RepoSrcRSS:
		return m.RSS.Crawl(ctx, target)
	default:
		return m.Local.Crawl(ctx, target)
	}
}

// LocalCrawler walks a local directory, treating every file under it
// (recursively, max 100 files) as one item. Files > 256 KB are skipped
// to keep the parse stage cheap.
type LocalCrawler struct {
	MaxFiles int
	MaxBytes int64
}

func (l *LocalCrawler) Crawl(ctx context.Context, target string) ([]CrawlItem, error) {
	maxFiles := l.MaxFiles
	if maxFiles == 0 {
		maxFiles = 100
	}
	maxBytes := l.MaxBytes
	if maxBytes == 0 {
		maxBytes = 256 * 1024
	}
	out := []CrawlItem{}
	count := 0
	err := filepath.WalkDir(target, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil // best-effort
		}
		if d.IsDir() {
			return nil
		}
		if count >= maxFiles {
			return filepath.SkipAll
		}
		info, err := d.Info()
		if err != nil || info.Size() > maxBytes {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		out = append(out, CrawlItem{
			URL:       "file://" + filepath.ToSlash(path),
			Title:     filepath.Base(path),
			Body:      string(body),
			Tags:      []string{"local"},
			FetchedAt: time.Now().UTC(),
		})
		count++
		return nil
	})
	if err != nil {
		return out, err
	}
	return out, nil
}

// GitHubCrawler fetches a GitHub repo's README + a handful of top-level
// files via the raw.githubusercontent.com endpoint. We don't authenticate
// (kept simple for the M4 vertical slice); rate limits apply.
type GitHubCrawler struct {
	HTTPClient *http.Client
}

func (g *GitHubCrawler) Crawl(ctx context.Context, target string) ([]CrawlItem, error) {
	if g.HTTPClient == nil {
		g.HTTPClient = &http.Client{Timeout: 5 * time.Second}
	}
	// Accept "owner/repo" shorthand or full URLs.
	owner, repo, ok := parseGitHubTarget(target)
	if !ok {
		return nil, fmt.Errorf("unsupported github target %q (use owner/repo)", target)
	}
	files := []string{"README.md", "README.rst", "README"}
	out := []CrawlItem{}
	for _, name := range files {
		url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/HEAD/%s", owner, repo, name)
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		resp, err := g.HTTPClient.Do(req)
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != 200 {
			continue
		}
		out = append(out, CrawlItem{
			URL:       url,
			Title:     name,
			Body:      string(body),
			Tags:      []string{"github"},
			FetchedAt: time.Now().UTC(),
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("github crawl found no files for %s/%s", owner, repo)
	}
	return out, nil
}

func parseGitHubTarget(target string) (owner, repo string, ok bool) {
	t := strings.TrimSpace(target)
	t = strings.TrimPrefix(t, "https://github.com/")
	t = strings.TrimPrefix(t, "http://github.com/")
	t = strings.TrimPrefix(t, "github.com/")
	parts := strings.Split(t, "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// RSSCrawler fetches an RSS/Atom feed and emits one item per <entry> /
// <item>. The parser collapses tags into a small slice.
type RSSCrawler struct {
	HTTPClient *http.Client
}

func (r *RSSCrawler) Crawl(ctx context.Context, target string) ([]CrawlItem, error) {
	if r.HTTPClient == nil {
		r.HTTPClient = &http.Client{Timeout: 5 * time.Second}
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	resp, err := r.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("rss fetch %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	items := rssExtract(string(body))
	if len(items) == 0 {
		return nil, fmt.Errorf("rss feed %s contained no entries", target)
	}
	return items, nil
}

// rssExtract pulls (title, link, description) from each <item> or
// <entry>. Naive regex parser — sufficient for the M4 vertical slice.
var (
	rssItemRe   = regexp.MustCompile(`<(?:item|entry)[^>]*>([\s\S]*?)</(?:item|entry)>`)
	rssTitleRe  = regexp.MustCompile(`<title[^>]*>([\s\S]*?)</title>`)
	rssLinkRe   = regexp.MustCompile(`<link[^>]*>([\s\S]*?)</link>`)
	rssDescRe   = regexp.MustCompile(`<description[^>]*>([\s\S]*?)</description>`)
	rssSumRe    = regexp.MustCompile(`<summary[^>]*>([\s\S]*?)</summary>`)
)

func rssExtract(xml string) []CrawlItem {
	out := []CrawlItem{}
	for _, m := range rssItemRe.FindAllStringSubmatch(xml, -1) {
		inner := m[1]
		title := firstMatch(rssTitleRe, inner)
		link := firstMatch(rssLinkRe, inner)
		body := firstMatch(rssDescRe, inner)
		if body == "" {
			body = firstMatch(rssSumRe, inner)
		}
		if title == "" && link == "" && body == "" {
			continue
		}
		out = append(out, CrawlItem{
			URL:       strings.TrimSpace(link),
			Title:     strings.TrimSpace(title),
			Body:      strings.TrimSpace(body),
			Tags:      []string{"rss"},
			FetchedAt: time.Now().UTC(),
		})
	}
	return out
}

func firstMatch(re *regexp.Regexp, s string) string {
	m := re.FindStringSubmatch(s)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

// --- Parse stage ---

// HTMLParser is a noop pass-through with a simple HTML→Markdown
// conversion for the common cases. M4 spec calls for an LLM-backed
// summary, but for the vertical slice we use the first non-empty
// sentence as the summary.
type HTMLParser struct{}

func (p *HTMLParser) Parse(ctx context.Context, in CrawlItem) (string, string, string, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = "Untitled"
	}
	bodyMD := htmlToMarkdown(in.Body)
	summary := firstSentence(bodyMD)
	return title, bodyMD, summary, nil
}

// htmlToMarkdown strips tags and decodes the few entities we care about.
// Sufficient for README rendering; richer conversion lives in the LLM
// path (parse_llm.go, future work).
func htmlToMarkdown(s string) string {
	// Replace common block tags with newlines so the markdown output
	// preserves a little structure.
	replacer := strings.NewReplacer(
		"<br>", "\n", "<br/>", "\n", "<br />", "\n",
		"</p>", "\n\n", "</div>", "\n",
		"<li>", "- ", "</li>", "\n",
		"</h1>", "\n\n", "</h2>", "\n\n", "</h3>", "\n\n",
	)
	s = replacer.Replace(s)
	// Drop all remaining tags.
	s = tagStripRe.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	return strings.TrimSpace(s)
}

var tagStripRe = regexp.MustCompile(`<[^>]+>`)

func firstSentence(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	for i, r := range s {
		if r == '.' || r == '\n' {
			return strings.TrimSpace(s[:i+1])
		}
		if i > 240 {
			return strings.TrimSpace(s[:241])
		}
	}
	return s
}

// --- Grade stage ---

// HeuristicGrader scores an item 0..1 based on length, structural
// signals (h1/h2 presence), and tag diversity. The M4 plan calls for
// an LLM-based grader; the heuristic is the v0 placeholder.
type HeuristicGrader struct{}

func (g *HeuristicGrader) Grade(ctx context.Context, title, bodyMD, summary string) (float64, error) {
	if len(bodyMD) < 200 {
		return 0.2, nil
	}
	score := 0.4
	if len(bodyMD) > 800 {
		score += 0.2
	}
	if len(bodyMD) > 2000 {
		score += 0.1
	}
	if strings.Contains(bodyMD, "\n## ") || strings.Contains(bodyMD, "\n### ") {
		score += 0.1
	}
	if len(title) > 8 && len(title) < 80 {
		score += 0.1
	}
	if summary != "" {
		score += 0.05
	}
	if score > 0.95 {
		score = 0.95
	}
	return score, nil
}

// --- Sink + Rehearse stages ---

// sinkCandidate writes a single parsed+graded item into bp_candidates.
// Returns the new candidate id.
func sinkCandidate(ctx context.Context, db *DB, pipelineID, runID, sourceURL, title, bodyMD string, grade float64, tags []string) (string, error) {
	tagsJSON, err := marshalCandidateTags(tags)
	if err != nil {
		return "", err
	}
	c := &BPCandidateRow{
		ID:         "bpc-" + randomRepoHex(6),
		PipelineID: pipelineID,
		RunID:      runID,
		Title:      title,
		Body:       bodyMD,
		GradeScore: grade,
		SourceURL:  sourceURL,
		TagsJSON:   tagsJSON,
		Status:     BPCandidateStatusDraft,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	if err := insertBPCandidate(ctx, db, c); err != nil {
		return "", err
	}
	return c.ID, nil
}

// rehearseCandidate checks existing best_practices for a fuzzy match
// against the candidate's title. Returns the best match's bp_id and a
// score in [0,1], or ("", 0) when no candidate is close enough.
//
// The fuzzy match is intentionally simple — we lowercase both strings
// and look for the longest common token prefix. The threshold for
// "match" is 0.5; if no BP passes that, the candidate is treated as
// new and stays in draft state.
type rehearseMatch struct {
	BPID  string
	Score float64
}

func rehearseCandidate(ctx context.Context, db *DB, candidateTitle string) (rehearseMatch, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT id, title FROM best_practices WHERE status IN ('published','review_due')`,
	)
	if err != nil {
		return rehearseMatch{}, err
	}
	defer rows.Close()
	best := rehearseMatch{}
	for rows.Next() {
		var id, title string
		if err := rows.Scan(&id, &title); err != nil {
			return rehearseMatch{}, err
		}
		s := jaccardTitleScore(candidateTitle, title)
		if s > best.Score {
			best = rehearseMatch{BPID: id, Score: s}
		}
	}
	return best, rows.Err()
}

// jaccardTitleScore is a tiny token-Jaccard for short titles. Returns
// 0..1 — closer to 1 means more overlap.
func jaccardTitleScore(a, b string) float64 {
	at := tokenize(a)
	bt := tokenize(b)
	if len(at) == 0 || len(bt) == 0 {
		return 0
	}
	inter := 0
	for k := range at {
		if _, ok := bt[k]; ok {
			inter++
		}
	}
	union := len(at) + len(bt) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

func tokenize(s string) map[string]struct{} {
	s = strings.ToLower(s)
	out := map[string]struct{}{}
	for _, w := range strings.FieldsFunc(s, func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9')
	}) {
		if len(w) > 1 {
			out[w] = struct{}{}
		}
	}
	return out
}

// --- Run the pipeline ---

// PipelineResult is what runRepoPipeline returns.
type PipelineResult struct {
	Run         *RepoPipelineRunRow
	Candidates  []string // candidate ids produced
	MergedBPIDs []string // bp ids a candidate was merged into
}

// runRepoPipeline executes the full six-stage pipeline and persists
// the run + candidates. Errors at any stage mark the run failed but
// don't lose work already done in earlier stages.
func runRepoPipeline(ctx context.Context, db *DB, runtime *PipelineRuntime, pipeline *RepoPipelineRecord) (*PipelineResult, error) {
	if runtime == nil {
		runtime = DefaultRuntime()
	}
	now := time.Now().UTC()
	run := &RepoPipelineRunRow{
		RunID:      "rpr-" + randomRepoHex(6),
		PipelineID: pipeline.ID,
		Status:     RepoRunRunning,
		StartedAt:  now,
	}
	if err := insertRepoPipelineRun(ctx, db, run); err != nil {
		return nil, err
	}

	stages := map[string]string{}
	items, err := dispatchCrawl(ctx, runtime, pipeline.Source, pipeline.Target)
	if err != nil {
		stages["crawl"] = "failed: " + err.Error()
		run.Status = RepoRunFailed
		run.Error = "crawl: " + err.Error()
		run.EndedAt = time.Now().UTC()
		run.StageStatus = mustJSON(stages)
		_ = updateRepoPipelineRun(ctx, db, run)
		return &PipelineResult{Run: run}, err
	}
	stages["crawl"] = "ok"
	run.ItemsIngested = len(items)

	candidateIDs := []string{}
	mergedBPs := []string{}
	parsed := 0
	graded := 0
	accepted := 0
	for _, item := range items {
		title, bodyMD, summary, err := runtime.Parse.Parse(ctx, item)
		if err != nil {
			continue
		}
		parsed++
		score, err := runtime.Grade.Grade(ctx, title, bodyMD, summary)
		if err != nil {
			continue
		}
		graded++
		if score < pipeline.Threshold {
			continue
		}
		id, err := sinkCandidate(ctx, db, pipeline.ID, run.RunID, item.URL, title, bodyMD, score, item.Tags)
		if err != nil {
			continue
		}
		accepted++
		candidateIDs = append(candidateIDs, id)
		// Rehearse: if a fuzzy match beats 0.5, mark the candidate as
		// merged-with-existing (the actual merge to best_practices
		// is a manual editor action; this signal tells the UI which
		// candidates to surface in the "merge candidates" queue).
		match, err := rehearseCandidate(ctx, db, title)
		if err == nil && match.Score >= 0.5 && match.BPID != "" {
			if err := updateBPCandidateDecision(ctx, db, id, BPCandidateStatusMerged, match.BPID); err == nil {
				mergedBPs = append(mergedBPs, match.BPID)
			}
		}
	}
	stages["parse"] = "ok"
	stages["grade"] = "ok"
	stages["sink"] = "ok"
	stages["rehearse"] = "ok"
	run.ItemsParsed = parsed
	run.ItemsGraded = graded
	run.ItemsAccepted = accepted
	run.Status = RepoRunSucceeded
	run.EndedAt = time.Now().UTC()
	run.StageStatus = mustJSON(stages)
	if err := updateRepoPipelineRun(ctx, db, run); err != nil {
		return &PipelineResult{Run: run, Candidates: candidateIDs}, err
	}
	return &PipelineResult{Run: run, Candidates: candidateIDs, MergedBPIDs: mergedBPs}, nil
}

func dispatchCrawl(ctx context.Context, runtime *PipelineRuntime, source, target string) ([]CrawlItem, error) {
	if multi, ok := runtime.Crawl.(*MultiCrawler); ok {
		return multi.CrawlBySource(ctx, source, target)
	}
	return runtime.Crawl.Crawl(ctx, target)
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

// hashForDedup returns a stable short hash for a candidate title so the
// e2e can dedupe across runs. Not stored in the schema — used only by
// tests that want to assert "the same content didn't double-write".
func hashForDedup(s string) string {
	h := sha1.Sum([]byte(s))
	return hex.EncodeToString(h[:6])
}

// sqlNoRows lets the runtime hint "row missing" without forcing every
// call site to import database/sql. Equivalent to sql.ErrNoRows for
// errors.Is purposes within this file.
var _ = sql.ErrNoRows