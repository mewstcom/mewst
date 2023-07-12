# typed: false
# frozen_string_literal: true

RSpec.describe "POST /internal/email_confirmations/:email_confirmation_id/challenge", type: :request, api_version: :internal do
  context "when invalid code" do
    let!(:email_confirmation) { create(:email_confirmation) }

    it "responses 422" do
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

      expect(response).to have_http_status(422)
      assert_response_schema_confirm(422)
    end
  end

  context "when valid code" do
    let!(:email_confirmation) { create(:email_confirmation) }

    it "responses 201" do
      expect(EmailConfirmation.count).to eq(1)
      expect(email_confirmation.succeeded_at).to be_nil

      post("/internal/email_confirmations/#{email_confirmation.id}/challenge", params: {
        confirmation_code: email_confirmation.code
      })

      expect(EmailConfirmation.count).to eq(1)
      email_confirmation = EmailConfirmation.first
      expect(email_confirmation.succeeded_at).to_not be_nil

      expect(response).to have_http_status(204)
      assert_response_schema_confirm(204)
    end
  end
end
