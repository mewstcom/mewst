# typed: false
# frozen_string_literal: true

RSpec.describe "GET /internal/email_confirmations/:email_confirmation_id", type: :request, api_version: :internal do
  context "when unknown email_confirmation_id" do
    let!(:email_confirmation_id) { "unknown" }

    it "responses 404" do
      get "/internal/email_confirmations/#{email_confirmation_id}"

      expected = {
        errors: [
          {
            message: "Not found"
          }
        ]
      }
      actual = JSON.parse(response.body)
      expect(actual).to include(expected.deep_stringify_keys)

      expect(response).to have_http_status(:not_found)
      assert_response_schema_confirm(404)
    end
  end

  context "when existing email_confirmation_id" do
    let!(:email_confirmation) { create(:email_confirmation) }
    let!(:email_confirmation_id) { email_confirmation.id }

    it "responses 200" do
      get "/internal/email_confirmations/#{email_confirmation_id}"

      expected = {
        email_confirmation: {
          email: email_confirmation.email,
          id: email_confirmation.id,
          succeeded_at: nil
        }
      }
      actual = JSON.parse(response.body)
      expect(actual).to include(expected.deep_stringify_keys)

      expect(response).to have_http_status(:ok)
      assert_response_schema_confirm(200)
    end
  end
end
