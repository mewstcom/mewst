package usecase

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/validator"
)

const (
	// maxLinkRedirects bounds how many redirects fetchHTML follows. Rails has
	// no explicit bound (a mutual redirect loop between allowed domains would
	// recurse forever), so this is an intentional safety addition on the Go side.
	//
	// [Ja] maxLinkRedirects は fetchHTML が追跡するリダイレクト回数の上限。Rails 側には
	// 明示的な上限が無く (許可ドメイン同士の相互リダイレクトで無限再帰しうる)、Go 側で
	// 意図的に追加した安全策。
	maxLinkRedirects = 5

	// maxLinkHTMLBytes caps how many bytes of the fetched HTML are read, so a
	// huge (or maliciously endless) response cannot exhaust memory. The OGP
	// metadata we need lives in <head>, so 5 MiB is more than enough.
	//
	// [Ja] maxLinkHTMLBytes は取得した HTML の読み込みバイト数の上限。巨大な (あるいは
	// 悪意ある無限長の) レスポンスでメモリを使い果たさないようにする。必要な OGP
	// メタデータは <head> 内にあるため 5 MiB あれば十分。
	maxLinkHTMLBytes = 5 << 20
)

// redirectAllowedDomains maps a domain to an extra domain it is allowed to
// redirect to. Redirects are otherwise only followed within the same domain,
// to avoid fetching metadata from a phishing destination (mirrors Rails
// LinkDataFetcher::REDIRECT_ALLOWED_DOMAINS).
//
// [Ja] redirectAllowedDomains はドメインごとにリダイレクトを許可する追加ドメインの
// 対応表。これ以外のリダイレクトは同一ドメイン内のみ追跡し、フィッシング先の
// メタデータを取得しないようにする (Rails の LinkDataFetcher::REDIRECT_ALLOWED_DOMAINS
// に対応)。
var redirectAllowedDomains = map[string]string{
	"youtu.be": "youtube.com",
}

// FetchLinkMetadataUsecase fetches the metadata of a target URL to build a
// link card, mirroring the Rails LinkDataFetcher + CreateLinkUseCase: it
// validates the target URL, reuses an existing link when one matches the
// canonical URL, otherwise fetches the page, extracts canonical_url / domain /
// title / image_url from the HTML, and persists a new link.
//
// [Ja] FetchLinkMetadataUsecase はリンクカードを作るために対象 URL のメタデータを
// 取得するオーケストレーション UseCase で、Rails の LinkDataFetcher + CreateLinkUseCase
// に対応する。対象 URL をバリデーションし、canonical URL が一致する既存リンクが
// あれば再利用、無ければページを取得して HTML から canonical_url / domain / title /
// image_url を抽出し、新しいリンクとして永続化する。
type FetchLinkMetadataUsecase struct {
	linkValidator *validator.LinkDataFetcherValidator
	linkRepo      *repository.LinkRepository
	httpClient    *http.Client

	// blockPrivateHosts refuses fetching from hosts that resolve to a private /
	// loopback / link-local address, to prevent SSRF against internal services.
	// It is configurable so tests (which talk to httptest servers on loopback)
	// can disable the block; production wiring passes true.
	//
	// [Ja] blockPrivateHosts は private / loopback / link-local アドレスに解決される
	// ホストからの取得を拒否し、内部サービスへの SSRF を防ぐ。テスト (loopback 上の
	// httptest サーバーと通信する) ではブロックを無効化できるよう設定可能にしている。
	// 本番配線では true を渡す。
	blockPrivateHosts bool
}

// NewFetchLinkMetadataUsecase は FetchLinkMetadataUsecase を生成する
func NewFetchLinkMetadataUsecase(
	linkValidator *validator.LinkDataFetcherValidator,
	linkRepo *repository.LinkRepository,
	httpClient *http.Client,
	blockPrivateHosts bool,
) *FetchLinkMetadataUsecase {
	// Copy the client and disable automatic redirects: the redirect policy
	// (same domain or allowlisted domains only) is enforced manually in
	// fetchHTML, mirroring the Rails LinkDataFetcher.
	//
	// [Ja] クライアントをコピーして自動リダイレクトを無効化する。リダイレクトの方針
	// (同一ドメインか許可ドメインのみ追跡) は Rails の LinkDataFetcher に合わせて
	// fetchHTML 内で手動で適用する。
	client := *httpClient
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	return &FetchLinkMetadataUsecase{
		linkValidator:     linkValidator,
		linkRepo:          linkRepo,
		httpClient:        &client,
		blockPrivateHosts: blockPrivateHosts,
	}
}

// FetchLinkMetadataInput はリンクメタデータ取得の入力パラメータ
type FetchLinkMetadataInput struct {
	// TargetURL はリンクカードを生成する対象の URL
	TargetURL string
}

// FetchLinkMetadataOutput はリンクメタデータ取得の出力パラメータ
type FetchLinkMetadataOutput struct {
	Link *model.Link
}

// Execute resolves the target URL into a link: validation, reuse of an
// existing link, or fetch + parse + create.
//
// [Ja] Execute は対象 URL をリンクに解決する (バリデーション → 既存リンクの再利用 →
// 取得 + パース + 作成)。
func (uc *FetchLinkMetadataUsecase) Execute(ctx context.Context, input FetchLinkMetadataInput) (*FetchLinkMetadataOutput, error) {
	// 1. バリデーション
	if err := uc.linkValidator.Validate(ctx, validator.LinkDataFetcherValidatorInput{
		TargetURL: input.TargetURL,
	}); err != nil {
		return nil, err
	}

	// 2. Reuse an existing link whose canonical URL equals the target URL
	// without fetching (mirrors Rails LinkRecord.find_by(canonical_url:)).
	// [Ja] canonical URL が対象 URL と一致する既存リンクは取得せずに再利用する
	// (Rails の LinkRecord.find_by(canonical_url:) に対応)。
	link, err := uc.linkRepo.FindByCanonicalURL(ctx, input.TargetURL)
	if err != nil {
		return nil, fmt.Errorf("リンクの取得に失敗: %w", err)
	}
	if link != nil {
		return &FetchLinkMetadataOutput{Link: link}, nil
	}

	// 3. 取得・パース・(canonical 経由の再利用または) 作成
	link, err = uc.fetchAndCreateLink(ctx, input.TargetURL)
	if err != nil {
		return nil, err
	}

	return &FetchLinkMetadataOutput{Link: link}, nil
}

// fetchAndCreateLink fetches the target URL, extracts its metadata, reuses an
// existing link found via the page's canonical URL, and otherwise validates and
// persists a new link. It is split out of Execute so that Execute reads as pure
// orchestration (validate -> reuse by target URL -> fetch/create).
//
// [Ja] fetchAndCreateLink は対象 URL を取得してメタデータを抽出し、ページの canonical
// URL で見つかる既存リンクを再利用、無ければバリデーションして新しいリンクを永続化する。
// Execute がオーケストレーション (バリデーション → 対象 URL での再利用 → 取得・作成) として
// 読めるよう切り出している。
func (uc *FetchLinkMetadataUsecase) fetchAndCreateLink(ctx context.Context, targetURL string) (*model.Link, error) {
	// Fetch the HTML. Any failure yields an empty string, which surfaces as a
	// fetch error to the user (mirrors Rails rescuing Faraday::Error into "").
	// [Ja] HTML を取得する。失敗はすべて空文字列になり、ユーザーには取得エラーとして
	// 表示される (Rails が Faraday::Error を "" に rescue するのに対応)。
	htmlBody := uc.fetchHTML(ctx, targetURL, 0)
	if htmlBody == "" {
		return nil, newLinkFetchError(ctx)
	}

	meta := parseLinkMetadata(htmlBody, targetURL)

	// Reuse an existing link found via the page's canonical URL (a different
	// target URL may resolve to an already-known canonical URL).
	// [Ja] ページの canonical URL で見つかる既存リンクを再利用する (別の対象 URL が
	// 既知の canonical URL に解決されることがある)。
	if meta.CanonicalURL != targetURL {
		link, err := uc.linkRepo.FindByCanonicalURL(ctx, meta.CanonicalURL)
		if err != nil {
			return nil, fmt.Errorf("リンクの取得に失敗: %w", err)
		}
		if link != nil {
			return link, nil
		}
	}

	// Validate the fetched data like the Rails LinkForm; invalid metadata is
	// shown to the user as a fetch error.
	// [Ja] 取得データを Rails の LinkForm と同様にバリデーションする。不正なメタデータ
	// はユーザーには取得エラーとして表示される。
	if !meta.isValid() {
		return nil, newLinkFetchError(ctx)
	}

	// 永続化 (単一の insert のためトランザクションは開かない)
	link, err := uc.linkRepo.Create(ctx, repository.CreateLinkInput{
		CanonicalURL: meta.CanonicalURL,
		Domain:       meta.Domain,
		Title:        meta.Title,
		ImageURL:     meta.ImageURL,
	})
	if err != nil {
		return nil, fmt.Errorf("リンクの作成に失敗: %w", err)
	}

	return link, nil
}

// fetchHTML fetches the HTML of targetURL and returns "" on any failure. A
// redirect is followed only when its destination is the same domain (ignoring
// a "www." prefix) or an allowlisted domain, to avoid fetching metadata from a
// phishing destination; otherwise the redirect response body itself is parsed,
// mirroring the Rails LinkDataFetcher#fetch_html.
//
// [Ja] fetchHTML は targetURL の HTML を取得し、失敗時は "" を返す。リダイレクトは
// 遷移先が同一ドメイン ("www." プレフィックスは無視) か許可ドメインの場合のみ追跡し、
// フィッシング先のメタデータを取得しないようにする。それ以外はリダイレクトレスポンス
// 自体のボディをパース対象にする (Rails の LinkDataFetcher#fetch_html に対応)。
func (uc *FetchLinkMetadataUsecase) fetchHTML(ctx context.Context, targetURL string, redirectCount int) string {
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return ""
	}
	domain := strings.TrimPrefix(parsedURL.Hostname(), "www.")
	if domain == "" {
		return ""
	}

	// Refuse hosts that resolve to an internal address before issuing the
	// request, to prevent SSRF (e.g. cloud metadata endpoints, localhost
	// services). This runs on every redirect hop because fetchHTML recurses.
	// [Ja] リクエストを発行する前に、内部アドレスに解決されるホストを拒否して SSRF
	// (クラウドのメタデータエンドポイントや localhost のサービスなど) を防ぐ。fetchHTML
	// は再帰するため、各リダイレクトのホップでもこのチェックが効く。
	if uc.blockPrivateHosts && isPrivateHost(ctx, parsedURL.Hostname()) {
		return ""
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return ""
	}
	resp, err := uc.httpClient.Do(req)
	if err != nil {
		return ""
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 && resp.StatusCode <= 399 && redirectCount < maxLinkRedirects {
		// Resolve the Location header against the current URL so relative
		// redirects work (mirrors Rails URI.join).
		// [Ja] 相対リダイレクトも扱えるよう Location ヘッダーを現在の URL に対して
		// 解決する (Rails の URI.join に対応)。
		if redirectURL, err := parsedURL.Parse(resp.Header.Get("Location")); err == nil {
			redirectDomain := strings.TrimPrefix(redirectURL.Hostname(), "www.")
			if redirectDomain == domain || redirectDomain == redirectAllowedDomains[domain] {
				return uc.fetchHTML(ctx, redirectURL.String(), redirectCount+1)
			}
		}
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxLinkHTMLBytes))
	if err != nil {
		return ""
	}
	return string(body)
}

// isPrivateHost reports whether host resolves to a loopback / private /
// link-local / unspecified address. Such hosts are refused before fetching to
// prevent SSRF against internal services. A host that cannot be resolved (or
// resolves to no address) is also treated as disallowed, so fetching fails
// closed rather than open.
//
// [Ja] isPrivateHost は host が loopback / private / link-local / unspecified の
// アドレスに解決されるかを返す。これらのホストは取得前に拒否して、内部サービスへの
// SSRF を防ぐ。解決できない (またはアドレスが得られない) ホストも拒否扱いとし、fail
// open ではなく fail closed にする。
func isPrivateHost(ctx context.Context, host string) bool {
	if host == "" {
		return true
	}

	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil || len(ips) == 0 {
		return true
	}

	for _, addr := range ips {
		ip := addr.IP
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
			ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return true
		}
	}

	return false
}

// linkMetadata is the metadata extracted from a fetched page (the Rails
// LinkDataFetcher::FetchedData equivalent).
//
// [Ja] linkMetadata は取得したページから抽出したメタデータ (Rails の
// LinkDataFetcher::FetchedData に相当)。
type linkMetadata struct {
	CanonicalURL string
	Domain       string
	Title        string
	ImageURL     string
}

// isValid mirrors the Rails LinkForm validation: canonical_url must be a valid
// URL, domain and title must be present, and image_url must be a valid URL
// when present.
//
// [Ja] isValid は Rails の LinkForm のバリデーションに対応する。canonical_url は
// 有効な URL、domain と title は必須、image_url は値がある場合のみ有効な URL で
// あること。
func (m linkMetadata) isValid() bool {
	if !validator.IsValidURL(m.CanonicalURL) {
		return false
	}
	if m.Domain == "" || m.Title == "" {
		return false
	}
	if m.ImageURL != "" && !validator.IsValidURL(m.ImageURL) {
		return false
	}
	return true
}

// parseLinkMetadata extracts the link card metadata from the HTML, applying
// the same fallbacks as the Rails LinkDataFetcher#parse_html: the canonical
// URL falls back to the target URL, and the title falls back from og:title to
// <title> to the canonical URL.
//
// [Ja] parseLinkMetadata は HTML からリンクカードのメタデータを抽出する。フォール
// バックは Rails の LinkDataFetcher#parse_html に合わせ、canonical URL は対象 URL に、
// タイトルは og:title → <title> → canonical URL の順にフォールバックする。
func parseLinkMetadata(htmlBody, targetURL string) linkMetadata {
	tags := extractLinkTags(htmlBody)

	canonicalURL := firstPresent(tags.canonicalURL, targetURL)

	// Rails derives the domain with URI.parse(canonical_url).host. When the
	// canonical URL is unparsable the domain stays empty and isValid reports a
	// fetch error (Rails would raise instead; failing softly is safer here).
	//
	// [Ja] Rails は URI.parse(canonical_url).host でドメインを導出する。canonical URL
	// がパース不能な場合はドメインを空のままにし、isValid が取得エラーとして報告する
	// (Rails は例外になるが、ここでは穏当に失敗させるほうが安全)。
	domain := ""
	if u, err := url.Parse(canonicalURL); err == nil {
		domain = u.Hostname()
	}

	title := firstPresent(tags.ogTitle, tags.pageTitle, canonicalURL)

	return linkMetadata{
		CanonicalURL: canonicalURL,
		Domain:       domain,
		Title:        title,
		ImageURL:     tags.ogImage,
	}
}

// extractedLinkTags holds the raw tag values extracted from the HTML before
// fallbacks are applied.
//
// [Ja] extractedLinkTags はフォールバック適用前の、HTML から抽出した生のタグ値を
// 保持する。
type extractedLinkTags struct {
	canonicalURL string
	ogTitle      string
	ogImage      string
	pageTitle    string
}

// extractLinkTags walks the parsed HTML tree and picks the first occurrence of
// each tag the link card needs (link[rel=canonical], og:title, og:image and
// <title>), matching Nokogiri's at_css which returns the first match.
//
// [Ja] extractLinkTags はパース済み HTML ツリーを走査し、リンクカードに必要な各タグ
// (link[rel=canonical]・og:title・og:image・<title>) の最初の出現を拾う。最初の
// 一致を返す Nokogiri の at_css に合わせている。
func extractLinkTags(htmlBody string) extractedLinkTags {
	doc, err := html.Parse(strings.NewReader(htmlBody))
	if err != nil {
		return extractedLinkTags{}
	}

	var tags extractedLinkTags
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "link":
				if tags.canonicalURL == "" && nodeAttr(n, "rel") == "canonical" {
					tags.canonicalURL = nodeAttr(n, "href")
				}
			case "meta":
				switch nodeAttr(n, "property") {
				case "og:title":
					if tags.ogTitle == "" {
						tags.ogTitle = nodeAttr(n, "content")
					}
				case "og:image":
					if tags.ogImage == "" {
						tags.ogImage = nodeAttr(n, "content")
					}
				}
			case "title":
				if tags.pageTitle == "" {
					tags.pageTitle = textContent(n)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return tags
}

// nodeAttr returns the value of the named attribute, or "" when absent.
// [Ja] nodeAttr は指定した属性の値を返す (無ければ "")。
func nodeAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

// textContent concatenates the direct text children of the node.
// [Ja] textContent はノード直下のテキスト子ノードを連結して返す。
func textContent(n *html.Node) string {
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			sb.WriteString(c.Data)
		}
	}
	return sb.String()
}

// firstPresent returns the first value that is not blank (empty or whitespace
// only), mirroring Rails' `.presence ||` fallback chains.
//
// [Ja] firstPresent は blank (空または空白のみ) でない最初の値を返す。Rails の
// `.presence ||` によるフォールバックに対応する。
func firstPresent(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// newLinkFetchError returns the validation error shown when link information
// cannot be fetched from the target URL (the Rails
// LinkDataFetcherForm#add_fetch_error! equivalent).
//
// [Ja] newLinkFetchError は対象 URL からリンク情報を取得できなかったときに表示する
// バリデーションエラーを返す (Rails の LinkDataFetcherForm#add_fetch_error! に相当)。
func newLinkFetchError(ctx context.Context) *model.ValidationError {
	ve := model.NewValidationError()
	ve.AddField("target_url", i18n.T(ctx, "validation_link_fetch_failed"))
	return ve
}
