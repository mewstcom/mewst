# typed: false
# frozen_string_literal: true

RSpec.describe LinkDataFetcher do
  describe ".call" do
    it "全てのデータが取得できるとき、全てのデータを返すこと" do
      target_url = "https://example.com?foo=bar"

      VCR.use_cassette("link_data_fetcher/valid") do
        result = LinkDataFetcher.call(target_url:)

        expect(result.link).to be_nil
        expect(result.fetched_data.canonical_url).to eq("https://example.com")
        expect(result.fetched_data.domain).to eq("example.com")
        expect(result.fetched_data.title).to eq("Example Domain")
        expect(result.fetched_data.image_url).to eq("https://example.com/image.jpg")
      end
    end
  end

  describe "#fetch_html" do
    it "wwwありからwwwなしにリダイレクトするとき、データが取得できること" do
      target_url = "https://www.annict.com"

      VCR.use_cassette("link_data_fetcher/redirect_www_to_no_www") do
        fetcher = LinkDataFetcher.new(target_url:)
        html = fetcher.fetch_html
        binding.irb

        expect(result.link).to be_nil
        expect(result.fetched_data.canonical_url).to eq("https://annict.com/")
        expect(result.fetched_data.domain).to eq("annict.com")
        expect(result.fetched_data.title).to eq("Annict | 見たアニメを記録して、共有しよう")
        expect(result.fetched_data.image_url).to eq("https://annict.com/images/og_image.png")
      end
    end

    context "別ドメインにリダイレクトするとき" do
      it "リダイレクト前のデータを取得すること" do
        target_url = "https://www.publickey.jp"

        VCR.use_cassette("link_data_fetcher/redirect_other_domain") do
          result = LinkDataFetcher.new(target_url:).call

          expect(result.link).to be_nil
          expect(result.fetched_data.canonical_url).to eq("https://www.publickey.jp")
          expect(result.fetched_data.domain).to eq("www.publickey.jp")
          expect(result.fetched_data.title).to eq("301 Moved Permanently")
          expect(result.fetched_data.image_url).to be_nil
        end
      end
    end
  end

  describe "#parse_html" do
    context "canonicalタグがないとき" do
      it "`target_url` が `canonical_url` となること" do
        target_url = "https://example.com?foo=bar"

        VCR.use_cassette("link_data_fetcher/without_canonical") do
          result = LinkDataFetcher.new(target_url:).call

          expect(result.link).to be_nil
          expect(result.fetched_data.canonical_url).to eq(target_url)
        end
      end
    end

    context "og:imageタグがないとき" do
      it "`image_url` が `nil` となること" do
        target_url = "https://example.com"

        VCR.use_cassette("link_data_fetcher/without_og_image") do
          result = LinkDataFetcher.new(target_url:).call

          expect(result.link).to be_nil
          expect(result.fetched_data.image_url).to be_nil
        end
      end
    end

    context "titleタグがないとき" do
      it "`title` が `target_url` となること" do
        target_url = "https://example.com"

        VCR.use_cassette("link_data_fetcher/without_title") do
          result = LinkDataFetcher.new(target_url:).call

          expect(result.link).to be_nil
          expect(result.fetched_data.title).to eq(target_url)
        end
      end
    end
  end
end
