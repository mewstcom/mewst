# typed: strict
# frozen_string_literal: true

class LinkDataFetcher
  extend T::Sig

  class FetchedData < T::Struct
    const :canonical_url, String
    const :domain, String
    const :title, String
    const :image_url, T.nilable(String)
  end

  class Result < T::Struct
    const :link, T.nilable(LinkRecord)
    const :fetched_data, T.nilable(FetchedData)
  end

  REDIRECT_ALLOWED_DOMAINS = T.let({
    "youtu.be" => "youtube.com"
  }.freeze, T::Hash[String, String])

  sig { params(target_url: String).returns(Result) }
  def call(target_url:)
    saved_link = LinkRecord.find_by(canonical_url: target_url)
    return Result.new(link: saved_link) if saved_link

    html = fetch_html(target_url:)
    return Result.new if html.blank?

    parse_html(html:, target_url:)
  end

  sig { params(target_url: String).returns(String) }
  private def fetch_html(target_url:)
    domain = URI.parse(target_url).host&.delete_prefix("www.")
    return "" unless domain

    response = Faraday.get(target_url)

    # リダイレクトを検知し、同じドメインか許可している別ドメインの場合はリダイレクト先の情報を取得する
    # フィッシングに利用される可能性があるため、他の許可していないドメインにリダイレクトされる場合は情報を取得しない
    if response.status.between?(300, 399)
      location = response.headers["location"]
      redirect_url = URI.join(target_url, location).to_s
      redirect_domain = URI.parse(redirect_url).host&.delete_prefix("www.")

      if redirect_domain&.in?(allowed_domains(domain:))
        return fetch_html(target_url: redirect_url)
      end
    end

    response.body.force_encoding("UTF-8")
  rescue Faraday::Error
    ""
  end

  sig { params(html: String, target_url: String).returns(Result) }
  private def parse_html(html:, target_url:)
    doc = Nokogiri::HTML(html)
    fetched_canonical_url = doc.at_css('link[rel="canonical"]')&.[]("href")

    if fetched_canonical_url
      saved_link = LinkRecord.find_by(canonical_url: fetched_canonical_url)
      return Result.new(link: saved_link) if saved_link
    end

    canonical_url = fetched_canonical_url.presence || target_url
    domain = URI.parse(canonical_url).host.not_nil!
    title = doc.at_css('meta[property="og:title"]')&.[]("content").presence ||
      doc.at_css("title")&.text.presence ||
      canonical_url
    image_url = doc.at_css('meta[property="og:image"]')&.[]("content")

    Result.new(fetched_data: FetchedData.new(canonical_url:, domain:, title:, image_url:))
  end

  sig { params(domain: String).returns(T::Array[String]) }
  private def allowed_domains(domain:)
    redirect_domain = REDIRECT_ALLOWED_DOMAINS[domain]
    redirect_domain ? [domain, redirect_domain] : [domain]
  end
end
