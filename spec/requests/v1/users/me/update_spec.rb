# typed: false
# frozen_string_literal: true

# rubocop:disable RSpec/EmptyExampleGroup
RSpec.xdescribe "PATCH /v1/users/me", type: :request, api_version: :v1 do
  # context "言語が不正なとき" do
  #   let!(:viewer) { create(:actor, :with_access_token_for_web) }
  #   let!(:oauth_access_token) { viewer.oauth_access_tokens.first }
  #   let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }

  #   it "`422` を返すこと" do
  #     patch("/v1/users/me", headers:, params: {
  #       locale: "unknown-locale",
  #       time_zone: "Asia/Tokyo"
  #     })
  #     expect(response).to have_http_status(:unprocessable_entity)

  #     expected = {
  #       errors: [
  #         {
  #           code: "invalid_input_data",
  #           field: "locale",
  #           message: "Locale is not included in the list"
  #         }
  #       ]
  #     }
  #     actual = JSON.parse(response.body)
  #     expect(actual).to include(expected.deep_stringify_keys)

  #     assert_response_schema_confirm(422)
  #   end
  # end

  # context "入力データが正しいとき" do
  #   let!(:viewer) { create(:actor, :with_access_token_for_web) }
  #   let!(:user) { viewer.user_record }
  #   let!(:oauth_access_token) { viewer.oauth_access_tokens.first }
  #   let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }

  #   before do
  #     user.update!(locale: "en")
  #   end

  #   it "`200` を返すこと" do
  #     expect(user.locale).to eq("en")

  #     patch("/v1/users/me", headers:, params: {
  #       locale: "ja",
  #       time_zone: "Asia/Tokyo"
  #     })
  #     expect(response).to have_http_status(:ok)

  #     expect(User.count).to eq(1)
  #     updated_user = User.first

  #     expect(updated_user.locale).to eq("ja")

  #     expected = {
  #       user: build_user_resource(user: updated_user)
  #     }
  #     actual = JSON.parse(response.body)
  #     expect(actual).to include(expected.deep_stringify_keys)

  #     assert_response_schema_confirm(200)
  #   end
  # end
end
# rubocop:enable RSpec/EmptyExampleGroup
