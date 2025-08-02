# typed: false
# frozen_string_literal: true

# rubocop:disable RSpec/EmptyExampleGroup
RSpec.xdescribe "POST /v1/internal/email_confirmations/:email_confirmation_id/challenge", type: :request do
  # context "確認用コードが不正なとき" do
  #   let!(:token) { ActionController::HttpAuthentication::Token.encode_credentials(Rails.configuration.mewst["internal_api_token"]) }
  #   let!(:headers) { {"HTTP_AUTHORIZATION" => token} }
  #   let!(:email_confirmation) { create(:email_confirmation_record) }

  #   it "`422` を返すこと" do
  #     post("/v1/internal/email_confirmations/#{email_confirmation.id}/challenge", headers:, params: {
  #       confirmation_code: "invalid_code"
  #     })

  #     expected = {
  #       errors: [
  #         {
  #           code: "invalid_input_data",
  #           field: "confirmation_code",
  #           message: "Confirmation code is incorrect or expired"
  #         }
  #       ]
  #     }
  #     actual = JSON.parse(response.body)
  #     expect(actual).to include(expected.deep_stringify_keys)

  #     expect(response).to have_http_status(:unprocessable_entity)
  #   end
  # end

  # context "確認用コードが正しいとき" do
  #   let!(:token) { ActionController::HttpAuthentication::Token.encode_credentials(Rails.configuration.mewst["internal_api_token"]) }
  #   let!(:headers) { {"HTTP_AUTHORIZATION" => token} }
  #   let!(:email_confirmation) { create(:email_confirmation_record) }

  #   it "`200` を返すこと" do
  #     expect(EmailConfirmation.count).to eq(1)
  #     expect(email_confirmation.succeeded_at).to be_nil

  #     post("/v1/internal/email_confirmations/#{email_confirmation.id}/challenge", headers:, params: {
  #       confirmation_code: email_confirmation.code
  #     })

  #     expect(EmailConfirmation.count).to eq(1)
  #     email_confirmation = EmailConfirmation.first
  #     expect(email_confirmation.succeeded_at).not_to be_nil

  #     expected = {
  #       email_confirmation: build_email_confirmation_resource(email_confirmation:)
  #     }
  #     actual = JSON.parse(response.body)
  #     expect(actual).to include(expected.deep_stringify_keys)

  #     expect(response).to have_http_status(:ok)
  #   end
  # end
end
# rubocop:enable RSpec/EmptyExampleGroup
