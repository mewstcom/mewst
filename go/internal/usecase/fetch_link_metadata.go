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
	// maxLinkRedirectsはfetchHTMLが追跡するリダイレクト回数の上限。Rails側には
	// 明示的な上限が無く (許可ドメイン同士の相互リダイレクトで無限再帰しうる)、Go側で
	// 意図的に追加した安全策。
	maxLinkRedirects = 5

	// maxLinkHTMLBytesは取得したHTMLの読み込みバイト数の上限。巨大な (あるいは
	// 悪意ある無限長の) レスポンスでメモリを使い果たさないようにする。必要なOGP
	// メタデータは <head> 内にあるため5 MiBあれば十分。
	maxLinkHTMLBytes = 5 << 20
)

// redirectAllowedDomainsはドメインごとにリダイレクトを許可する追加ドメインの
// 対応表。これ以外のリダイレクトは同一ドメイン内のみ追跡し、フィッシング先の
// メタデータを取得しないようにする (RailsのLinkDataFetcher::REDIRECT_ALLOWED_DOMAINS
// に対応)。
var redirectAllowedDomains = map[string]string{
	"youtu.be": "youtube.com",
}

// FetchLinkMetadataUsecaseはリンクカードを作るために対象URLのメタデータを
// 取得するオーケストレーションUseCaseで、RailsのLinkDataFetcher + CreateLinkUseCase
// に対応する。対象URLをバリデーションし、canonical URLが一致する既存リンクが
// あれば再利用、無ければページを取得してHTMLからcanonical_url / domain / title /
// image_urlを抽出し、新しいリンクとして永続化する。
type FetchLinkMetadataUsecase struct {
	linkValidator *validator.LinkDataFetcherValidator
	linkRepo      *repository.LinkRepository
	httpClient    *http.Client

	// blockPrivateHostsはprivate / loopback / link-localアドレスに解決される
	// ホストからの取得を拒否し、内部サービスへのSSRFを防ぐ。テスト (loopback上の
	// httptestサーバーと通信する) ではブロックを無効化できるよう設定可能にしている。
	// 本番配線ではtrueを渡す。
	blockPrivateHosts bool
}

// NewFetchLinkMetadataUsecaseはFetchLinkMetadataUsecaseを生成する
func NewFetchLinkMetadataUsecase(
	linkValidator *validator.LinkDataFetcherValidator,
	linkRepo *repository.LinkRepository,
	httpClient *http.Client,
	blockPrivateHosts bool,
) *FetchLinkMetadataUsecase {
	// クライアントをコピーして自動リダイレクトを無効化する。リダイレクトの方針
	// (同一ドメインか許可ドメインのみ追跡) はRailsのLinkDataFetcherに合わせて
	// fetchHTML内で手動で適用する。
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

// FetchLinkMetadataInputはリンクメタデータ取得の入力パラメータ
type FetchLinkMetadataInput struct {
	// TargetURLはリンクカードを生成する対象のURL
	TargetURL string
}

// FetchLinkMetadataOutputはリンクメタデータ取得の出力パラメータ
type FetchLinkMetadataOutput struct {
	Link *model.Link
}

// Executeは対象URLをリンクに解決する (バリデーション → 既存リンクの再利用 →
// 取得 + パース + 作成)。
func (uc *FetchLinkMetadataUsecase) Execute(ctx context.Context, input FetchLinkMetadataInput) (*FetchLinkMetadataOutput, error) {
	// 1. バリデーション
	if err := uc.linkValidator.Validate(ctx, validator.LinkDataFetcherValidatorInput{
		TargetURL: input.TargetURL,
	}); err != nil {
		return nil, err
	}

	// canonical URLが対象URLと一致する既存リンクは取得せずに再利用する
	// (RailsのLinkRecord.find_by(canonical_url:) に対応)。
	link, err := uc.linkRepo.FindByCanonicalURL(ctx, input.TargetURL)
	if err != nil {
		return nil, fmt.Errorf("リンクの取得に失敗: %w", err)
	}
	if link != nil {
		return &FetchLinkMetadataOutput{Link: link}, nil
	}

	// 3. 取得・パース・(canonical経由の再利用または) 作成
	link, err = uc.fetchAndCreateLink(ctx, input.TargetURL)
	if err != nil {
		return nil, err
	}

	return &FetchLinkMetadataOutput{Link: link}, nil
}

// fetchAndCreateLinkは対象URLを取得してメタデータを抽出し、ページのcanonical
// URLで見つかる既存リンクを再利用、無ければバリデーションして新しいリンクを永続化する。
// Executeがオーケストレーション (バリデーション → 対象URLでの再利用 → 取得・作成) として
// 読めるよう切り出している。
func (uc *FetchLinkMetadataUsecase) fetchAndCreateLink(ctx context.Context, targetURL string) (*model.Link, error) {
	// HTMLを取得する。失敗はすべて空文字列になり、ユーザーには取得エラーとして
	// 表示される (RailsがFaraday::Errorを "" にrescueするのに対応)。
	htmlBody := uc.fetchHTML(ctx, targetURL, 0)
	if htmlBody == "" {
		return nil, newLinkFetchError(ctx)
	}

	meta := parseLinkMetadata(htmlBody, targetURL)

	// ページのcanonical URLで見つかる既存リンクを再利用する (別の対象URLが
	// 既知のcanonical URLに解決されることがある)。
	if meta.CanonicalURL != targetURL {
		link, err := uc.linkRepo.FindByCanonicalURL(ctx, meta.CanonicalURL)
		if err != nil {
			return nil, fmt.Errorf("リンクの取得に失敗: %w", err)
		}
		if link != nil {
			return link, nil
		}
	}

	// 取得データをRailsのLinkFormと同様にバリデーションする。不正なメタデータ
	// はユーザーには取得エラーとして表示される。
	if !meta.isValid() {
		return nil, newLinkFetchError(ctx)
	}

	// 永続化 (単一のinsertのためトランザクションは開かない)
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

// fetchHTMLはtargetURLのHTMLを取得し、失敗時は "" を返す。リダイレクトは
// 遷移先が同一ドメイン ("www." プレフィックスは無視) か許可ドメインの場合のみ追跡し、
// フィッシング先のメタデータを取得しないようにする。それ以外はリダイレクトレスポンス
// 自体のボディをパース対象にする (RailsのLinkDataFetcher#fetch_htmlに対応)。
func (uc *FetchLinkMetadataUsecase) fetchHTML(ctx context.Context, targetURL string, redirectCount int) string {
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return ""
	}
	domain := strings.TrimPrefix(parsedURL.Hostname(), "www.")
	if domain == "" {
		return ""
	}

	// リクエストを発行する前に、内部アドレスに解決されるホストを拒否してSSRF
	// (クラウドのメタデータエンドポイントやlocalhostのサービスなど) を防ぐ。fetchHTML
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
		// 相対リダイレクトも扱えるようLocationヘッダーを現在のURLに対して
		// 解決する (RailsのURI.joinに対応)。
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

// isPrivateHostはhostがloopback / private / link-local / unspecifiedの
// アドレスに解決されるかを返す。これらのホストは取得前に拒否して、内部サービスへの
// SSRFを防ぐ。解決できない (またはアドレスが得られない) ホストも拒否扱いとし、fail
// openではなくfail closedにする。
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

// linkMetadataは取得したページから抽出したメタデータ (Railsの
// LinkDataFetcher::FetchedDataに相当)。
type linkMetadata struct {
	CanonicalURL string
	Domain       string
	Title        string
	ImageURL     string
}

// isValidはRailsのLinkFormのバリデーションに対応する。canonical_urlは
// 有効なURL、domainとtitleは必須、image_urlは値がある場合のみ有効なURLで
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

// parseLinkMetadataはHTMLからリンクカードのメタデータを抽出する。フォール
// バックはRailsのLinkDataFetcher#parse_htmlに合わせ、canonical URLは対象URLに、
// タイトルはog:title → <title> → canonical URLの順にフォールバックする。
func parseLinkMetadata(htmlBody, targetURL string) linkMetadata {
	tags := extractLinkTags(htmlBody)

	canonicalURL := firstPresent(tags.canonicalURL, targetURL)

	// RailsはURI.parse(canonical_url).hostでドメインを導出する。canonical URL
	// がパース不能な場合はドメインを空のままにし、isValidが取得エラーとして報告する
	// (Railsは例外になるが、ここでは穏当に失敗させるほうが安全)。
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

// extractedLinkTagsはフォールバック適用前の、HTMLから抽出した生のタグ値を
// 保持する。
type extractedLinkTags struct {
	canonicalURL string
	ogTitle      string
	ogImage      string
	pageTitle    string
}

// extractLinkTagsはパース済みHTMLツリーを走査し、リンクカードに必要な各タグ
// (link[rel=canonical]・og:title・og:image・<title>) の最初の出現を拾う。最初の
// 一致を返すNokogiriのat_cssに合わせている。
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

// nodeAttrは指定した属性の値を返す (無ければ "")。
func nodeAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

// textContentはノード直下のテキスト子ノードを連結して返す。
func textContent(n *html.Node) string {
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			sb.WriteString(c.Data)
		}
	}
	return sb.String()
}

// firstPresentはblank (空または空白のみ) でない最初の値を返す。Railsの
// `.presence ||` によるフォールバックに対応する。
func firstPresent(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// newLinkFetchErrorは対象URLからリンク情報を取得できなかったときに表示する
// バリデーションエラーを返す (RailsのLinkDataFetcherForm#add_fetch_error! に相当)。
func newLinkFetchError(ctx context.Context) *model.ValidationError {
	ve := model.NewValidationError()
	ve.AddField("target_url", i18n.T(ctx, "validation_link_fetch_failed"))
	return ve
}
