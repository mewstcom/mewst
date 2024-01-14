# typed: false
# frozen_string_literal: true

RSpec.describe "POST /v1/internal/email_confirmations", type: :request do
  context "メールアドレスが不正なとき" do
    let!(:token) { ActionController::HttpAuthentication::Token.encode_credentials(Rails.configuration.mewst["internal_api_token"]) }
    let!(:headers) { {"HTTP_AUTHORIZATION" => token} }

    it "`422` を返すこと" do
      post("/v1/internal/email_confirmations", headers:, params: {
        email: "invalidemail.com",
        event: "sign_up"
      })

      expected = {
        errors: [
          {
            code: "invalid_input_data",
            field: "email",
            message: "Email is invalid"
          }
        ]
      }
      actual = JSON.parse(response.body)
      expect(actual).to include(expected.deep_stringify_keys)

      expect(response).to have_http_status(:unprocessable_entity)
    end
  end

  context "入力データが正しいとき" do
    let!(:token) { ActionController::HttpAuthentication::Token.encode_credentials(Rails.configuration.mewst["internal_api_token"]) }
    let!(:headers) { {"HTTP_AUTHORIZATION" => token} }

    it "`201` を返すこと" do
      expect(EmailConfirmation.count).to eq(0)

      post("/v1/internal/email_confirmations", headers:, params: {
        email: "shimbaco@example.com",
        event: "sign_up"
      })

      expect(EmailConfirmation.count).to eq(1)
      email_confirmation = EmailConfirmation.first

      expected = {
        email_confirmation: build_email_confirmation_resource(email_confirmation:)
      }
      actual = JSON.parse(response.body)
      expect(actual).to include(expected.deep_stringify_keys)

      expect(response).to have_http_status(:created)
    end
  end
end
