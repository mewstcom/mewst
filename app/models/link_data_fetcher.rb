# typed: strict
# frozen_string_literal: true

class LinkDataFetcher
  extend T::Sig

  class FetchedData < T::Struct
    const :canonical_url, T.nilable(String)
    const :domain, T.nilable(String)
    const :title, T.nilable(String)
    const :image_url, T.nilable(String)
  end

  class Result < T::Struct
    const :link, T.nilable(Link)
    const :fetched_data, T.nilable(FetchedData)
  end

  sig { params(target_url: String).returns(Result) }
  def self.call(target_url:)
    saved_link = Link.find_by(canonical_url: target_url)
    return Result.new(link: saved_link) if saved_link

    fetcher = new(target_url:)

    html = fetcher.fetch_html
    return Result.new if html.blank?

    fetcher.parse_html(html:)
  end

  sig { params(target_url: String).void }
  def initialize(target_url:)
    @target_url = target_url
  end

  sig { returns(String) }
  def fetch_html
    domain = URI.parse(target_url).host
    response = Faraday.get(target_url)

    # リダイレクトを検知し、同じドメインの場合はリダイレクト先の情報を取得する
    # フィッシングに利用される可能性があるため、他のドメインにリダイレクトされる場合はリダイレクト先の情報は取得しない
    if response.status.between?(300, 399)
      location = response.headers["location"]
      redirect_url = URI.join(target_url, location).to_s
      redirect_domain = URI.parse(redirect_url).host

      if redirect_domain&.delete_prefix("www.") == domain&.delete_prefix("www.")
        return fetch_data(url: redirect_url)
      end
    end

    response.body
  rescue Faraday::Error
    ""
  end

  sig { params(html: String).returns(Result) }
  def parse_html(html:)
    doc = Nokogiri::HTML(html)
    fetched_canonical_url = doc.at_css('link[rel="canonical"]')&.[]("href")

    if fetched_canonical_url
      saved_link = Link.find_by(canonical_url: fetched_canonical_url)
      return Result.new(link: saved_link) if saved_link
    end

    canonical_url = fetched_canonical_url.presence || target_url
    domain = URI.parse(canonical_url).host
    title = doc.at_css("title")&.text.presence || canonical_url
    image_url = doc.at_css('meta[property="og:image"]')&.[]("content")

    Result.new(fetched_data: FetchedData.new(canonical_url:, domain:, title:, image_url:))
  end

  sig { returns(String) }
  attr_reader :target_url
  private :target_url
end
