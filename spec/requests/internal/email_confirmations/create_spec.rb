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

      expect(response).to have_http_status(:unprocessable_entity)
      assert_response_schema_confirm(422)
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
          id: email_confirmation.id
        }
      }
      actual = JSON.parse(response.body)
      expect(actual).to include(expected.deep_stringify_keys)

      expect(response).to have_http_status(:created)
      assert_response_schema_confirm(201)
    end
  end
end
