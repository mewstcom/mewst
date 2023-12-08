# typed: false
# frozen_string_literal: true

RSpec.describe "POST /internal/email_confirmations/:email_confirmation_id/challenge", type: :request, api_version: :internal do
  context "確認コードが不正なとき" do
    let!(:email_confirmation) { create(:email_confirmation) }

    it "`422` を返すこと" do
      post("/internal/email_confirmations/#{email_confirmation.id}/challenge", params: {
        confirmation_code: "invalid_code"
      })

      expected = {
        errors: [
          {
            code: "invalid_input_data",
            field: "confirmation_code",
            message: "Confirmation code is incorrect or expired"
          }
        ]
      }
      actual = JSON.parse(response.body)
      expect(actual).to include(expected.deep_stringify_keys)

      expect(response).to have_http_status(:unprocessable_entity)
      assert_response_schema_confirm(422)
    end
  end

  context "確認コードが正しいとき" do
    let!(:email_confirmation) { create(:email_confirmation) }

    it "`200` を返すこと" do
      expect(EmailConfirmation.count).to eq(1)
      expect(email_confirmation.succeeded_at).to be_nil

      post("/internal/email_confirmations/#{email_confirmation.id}/challenge", params: {
        confirmation_code: email_confirmation.code
      })

      expect(EmailConfirmation.count).to eq(1)
      email_confirmation = EmailConfirmation.first
      expect(email_confirmation.succeeded_at).not_to be_nil

      expected = {
        email_confirmation: build_email_confirmation_resource(email_confirmation:)
      }
      actual = JSON.parse(response.body)
      expect(actual).to include(expected.deep_stringify_keys)

      expect(response).to have_http_status(:ok)
      assert_response_schema_confirm(200)
    end
  end
end
