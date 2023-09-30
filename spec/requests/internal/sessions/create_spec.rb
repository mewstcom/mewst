# typed: false
# frozen_string_literal: true

RSpec.describe "POST /internal/sessions", type: :request, api_version: :internal do
  context "パスワードが正しくないとき" do
    let!(:email) { "test@example.com" }
    let!(:password) { "correct_password" }
    let!(:user) { create(:user, email:, password:) }

    before do
      create(:actor, user:)
    end

    it "`422` を返すこと" do
      post("/internal/sessions", params: {
        email:,
        password: "incorrect_password"
      })

      expected = {
        errors: [
          {
            code: "invalid_input_data",
            field: "email_or_password",
            message: "Login failed. Email or password is incorrect"
          }
        ]
      }
      actual = JSON.parse(response.body)
      expect(actual).to include(expected.deep_stringify_keys)

      expect(response).to have_http_status(:unprocessable_entity)
      assert_response_schema_confirm(422)
    end
  end

  context "メールアドレスとパスワードが正しいとき" do
    let!(:email) { "test@example.com" }
    let!(:password) { "correct_password" }
    let!(:user) { create(:user, email:, password:) }
    let!(:actor) { create(:actor, :with_access_token_for_web, user:) }
    let!(:profile) { actor.profile }
    let!(:oauth_access_token) { actor.oauth_access_tokens.first }

    it "`201` を返すこと" do
      expect(user.sign_in_count).to eq(0)

      post("/internal/sessions", params: {
        email:,
        password:
      })
      user.reload

      expect(user.sign_in_count).to eq(1)

      expected = {
        account: {
          oauth_access_token: {
            token: oauth_access_token.token
          },
          profile: build_profile_resource(profile:, viewer_has_followed: false),
          user: {
            id: user.id,
            locale: user.locale
          }
        }
      }
      actual = JSON.parse(response.body)
      expect(actual).to include(expected.deep_stringify_keys)

      expect(response).to have_http_status(:created)
      assert_response_schema_confirm(201)
    end
  end
end
