# typed: false
# frozen_string_literal: true

RSpec.describe "POST /internal/email_confirmations", type: :request, api_version: :internal do
  context "when invalid email" do
    it "responses 422" do
      post("/internal/email_confirmations", params: {
        email: "invalidemail.com"
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

      assert_response_schema_confirm(422)
      expect(response).to have_http_status(422)
    end
  end

  context "when valid" do
    it "responses 201" do
      expect(EmailConfirmation.count).to eq(0)

      post("/internal/email_confirmations", params: {
        email: "shimbaco@example.com"
      })

      expect(EmailConfirmation.count).to eq(1)
      email_confirmation = EmailConfirmation.first

      expected = {
        email_confirmation: {
          code: email_confirmation.code,
          email: email_confirmation.email,
          event: email_confirmation.event
        }
      }
      actual = JSON.parse(response.body)
      expect(actual).to include(expected.deep_stringify_keys)

      assert_response_schema_confirm(201)
      expect(response).to have_http_status(201)
    end
  end
end
