# typed: false
# frozen_string_literal: true

RSpec.describe LinkDataFetcher do
  describe "#call" do
    it "全てのデータが取得できるとき、全てのデータを返すこと" do
      target_url = "https://example.com?foo=bar"

      VCR.use_cassette("link_data_fetcher/valid") do
        result = LinkDataFetcher.new.call(target_url:)

        expect(result.link).to be_nil
        expect(result.fetched_data.canonical_url).to eq("https://example.com")
        expect(result.fetched_data.domain).to eq("example.com")
        expect(result.fetched_data.title).to eq("Example Domain")
        expect(result.fetched_data.image_url).to eq("https://example.com/image.jpg")
      end
    end

    it "wwwありからwwwなしにリダイレクトするとき、HTMLが取得できること" do
      target_url = "https://www.annict.com"

      VCR.use_cassette("link_data_fetcher/redirect_www_to_no_www") do
        result = LinkDataFetcher.new.call(target_url:)

        expect(result.link).to be_nil
        expect(result.fetched_data.canonical_url).to eq("https://annict.com/")
        expect(result.fetched_data.domain).to eq("annict.com")
        expect(result.fetched_data.title).to eq("Annict | 見たアニメを記録して、共有しよう")
        expect(result.fetched_data.image_url).to eq("https://annict.com/images/og_image.png")
      end
    end

    it "別ドメインにリダイレクトするとき、リダイレクト前のデータを取得すること" do
      target_url = "https://www.publickey.jp"

      VCR.use_cassette("link_data_fetcher/redirect_other_domain") do
        result = LinkDataFetcher.new.call(target_url:)

        expect(result.link).to be_nil
        expect(result.fetched_data.canonical_url).to eq("https://www.publickey.jp")
        expect(result.fetched_data.domain).to eq("www.publickey.jp")
        expect(result.fetched_data.title).to eq("301 Moved Permanently")
        expect(result.fetched_data.image_url).to be_nil
      end
    end

    it "canonicalタグがないとき、`target_url` が `canonical_url` となること" do
      target_url = "https://example.com?foo=bar"

      VCR.use_cassette("link_data_fetcher/without_canonical") do
        result = LinkDataFetcher.new.call(target_url:)

        expect(result.link).to be_nil
        expect(result.fetched_data.canonical_url).to eq(target_url)
      end
    end

    it "og:imageタグがないとき、`image_url` が `nil` となること" do
      target_url = "https://example.com"

      VCR.use_cassette("link_data_fetcher/without_og_image") do
        result = LinkDataFetcher.new.call(target_url:)

        expect(result.link).to be_nil
        expect(result.fetched_data.image_url).to be_nil
      end
    end

    it "titleタグがないとき、`title` が `target_url` となること" do
      target_url = "https://example.com"

      VCR.use_cassette("link_data_fetcher/without_title") do
        result = LinkDataFetcher.new.call(target_url:)

        expect(result.link).to be_nil
        expect(result.fetched_data.title).to eq(target_url)
      end
    end

    it "`youtu.be` のURLが指定されたとき、`youtube.com` の情報を取得すること" do
      target_url = "https://youtu.be/pbQQAwSQUX4?si=XtT8QfSo5yM75-4b"

      VCR.use_cassette("link_data_fetcher/youtube_short_url") do
        result = LinkDataFetcher.new.call(target_url:)

        expect(result.link).to be_nil
        expect(result.fetched_data.canonical_url).to eq("https://www.youtube.com/watch?v=pbQQAwSQUX4")
        expect(result.fetched_data.domain).to eq("www.youtube.com")
        expect(result.fetched_data.title).to eq("【公式】『ちいかわ』第1話「かためのプリン／ホットケーキ」 - YouTube")
        expect(result.fetched_data.image_url).to eq("https://i.ytimg.com/vi/pbQQAwSQUX4/maxresdefault.jpg")
      end
    end

    it "ThreadsのURLが指定されたとき、情報が取得できること" do
      target_url = "https://www.threads.net/@mewstcom"

      VCR.use_cassette("link_data_fetcher/threads_url") do
        result = LinkDataFetcher.new.call(target_url:)

        expect(result.link).to be_nil
        expect(result.fetched_data.canonical_url).to eq("https://www.threads.net/@mewstcom")
        expect(result.fetched_data.domain).to eq("www.threads.net")
        expect(result.fetched_data.title).to eq("Mewst (ミュースト) (@mewstcom) • Threads, Say more")
        expect(result.fetched_data.image_url).to eq("https://scontent-itm1-1.cdninstagram.com/v/t51.2885-19/434521890_1126680301811525_7130821923454510285_n.jpg?stp=dst-jpg_s640x640_tt6&_nc_cat=104&ccb=1-7&_nc_sid=2a654c&_nc_ohc=djAffuo-75AQ7kNvgHosyr6&_nc_zt=24&_nc_ht=scontent-itm1-1.cdninstagram.com&oh=00_AYAz9Mp_tiH-wpziNLxvWT6wXWTtDpeqqFCcqSNZCmEngw&oe=67A94F5F")
      end
    end
  end
end
