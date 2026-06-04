# typed: false
# frozen_string_literal: true

# Lightweight stand-ins for `Sentry::ErrorEvent` and `Sentry::RequestInterface`.
# Using Struct keeps the spec free of SDK initialization coupling and verifies
# only the filter's behavior.
#
# [Ja] `Sentry::ErrorEvent` / `Sentry::RequestInterface` の軽量ダブル。
# Struct を使うことで SDK の初期化に依存せず、フィルタの挙動だけを検証する。
SentrySpecEventDouble = Struct.new(:request, keyword_init: true)
SentrySpecRequestDouble = Struct.new(:data, keyword_init: true)

RSpec.describe "Sentry PII masking" do # rubocop:disable RSpec/DescribeClass
  describe "config/initializers/filter_parameter_logging.rb" do
    subject(:filter) { ActiveSupport::ParameterFilter.new(Rails.application.config.filter_parameters) }

    let(:plain_value) { "user@example.com" }
    let(:filtered_value) { "[FILTERED]" }

    # `:email` is a substring filter, so derived keys such as `user_email` and
    # `new_email` are also covered. sentry-rails honors `filter_parameters`
    # when attaching request params to breadcrumbs, so this masking covers
    # both Rails logs and Sentry breadcrumbs.
    #
    # [Ja] `:email` は部分一致フィルタのため、`user_email` / `new_email` の
    # ような派生キーもカバーされる。sentry-rails は breadcrumbs にリクエスト
    # パラメータを添付する際に `filter_parameters` を尊重するため、この
    # マスクは Rails ログと Sentry breadcrumbs の両方に効く。
    [
      "email",
      "user_email",
      "new_email",
      "password",
      "token"
    ].each do |key|
      it "#{key} を [FILTERED] に置き換えること" do
        expect(filter.filter(key => plain_value)).to eq(key => filtered_value)
      end
    end

    it "良性のキーは変更しないこと" do
      benign = {"id" => 42, "content" => "hello"}
      expect(filter.filter(benign)).to eq(benign)
    end

    it "ネストしたハッシュの中の email もマスクすること" do
      payload = {"user" => {"email" => plain_value, "name" => "alice"}}
      expect(filter.filter(payload)).to eq("user" => {"email" => filtered_value, "name" => "alice"})
    end
  end

  describe "config/initializers/sentry.rb" do
    let(:config) { Sentry.configuration }

    describe "send_default_pii" do
      it "リクエストボディや Cookie の自動添付を無効化していること" do
        expect(config.send_default_pii).to be(false)
      end
    end

    describe "before_send" do
      let(:before_send) { config.before_send }

      it "lambda として登録されていること" do
        expect(before_send).to respond_to(:call)
      end

      it "email を [FILTERED] に置き換え、非センシティブキーはそのまま通すこと" do
        event = build_event(data: {"email" => "user@example.com", "username" => "alice"})

        result = before_send.call(event, {})

        expect(result.request.data["email"]).to eq("[FILTERED]")
        expect(result.request.data["username"]).to eq("alice")
      end

      it "password / password_confirmation を [FILTERED] に置き換えること" do
        event = build_event(data: {
          "password" => "super-secret",
          "password_confirmation" => "super-secret"
        })

        result = before_send.call(event, {})

        expect(result.request.data["password"]).to eq("[FILTERED]")
        expect(result.request.data["password_confirmation"]).to eq("[FILTERED]")
      end

      it "csrf_token / authenticity_token を [FILTERED] に置き換えること" do
        event = build_event(data: {
          "csrf_token" => "csrf-abc",
          "authenticity_token" => "auth-xyz"
        })

        result = before_send.call(event, {})

        expect(result.request.data["csrf_token"]).to eq("[FILTERED]")
        expect(result.request.data["authenticity_token"]).to eq("[FILTERED]")
      end

      it "turnstile_response / cf-turnstile-response を [FILTERED] に置き換えること" do
        event = build_event(data: {
          "turnstile_response" => "turnstile-abc",
          "cf-turnstile-response" => "cf-turnstile-xyz"
        })

        result = before_send.call(event, {})

        expect(result.request.data["turnstile_response"]).to eq("[FILTERED]")
        expect(result.request.data["cf-turnstile-response"]).to eq("[FILTERED]")
      end

      it "ネストしたハッシュ内の email もマスクすること" do
        event = build_event(data: {"user" => {"email" => "user@example.com"}})

        result = before_send.call(event, {})

        expect(result.request.data["user"]["email"]).to eq("[FILTERED]")
      end

      it "配列の中のハッシュに含まれる email もマスクすること" do
        event = build_event(data: {"members" => [{"email" => "a@example.com"}, {"email" => "b@example.com"}]})

        result = before_send.call(event, {})

        expect(result.request.data["members"][0]["email"]).to eq("[FILTERED]")
        expect(result.request.data["members"][1]["email"]).to eq("[FILTERED]")
      end

      it "配列の中のハッシュにさらにネストしたセンシティブキーもマスクすること" do
        event = build_event(data: {"items" => [{"user" => {"password" => "secret"}}]})

        result = before_send.call(event, {})

        expect(result.request.data["items"][0]["user"]["password"]).to eq("[FILTERED]")
      end

      it "request.data が nil でも例外にならないこと" do
        event = build_event(data: nil)

        expect { before_send.call(event, {}) }.not_to raise_error
      end

      it "request 自体がない event でも例外にならないこと" do
        event = SentrySpecEventDouble.new(request: nil)

        expect { before_send.call(event, {}) }.not_to raise_error
      end
    end

    def build_event(data:)
      SentrySpecEventDouble.new(request: SentrySpecRequestDouble.new(data: data))
    end
  end
end
