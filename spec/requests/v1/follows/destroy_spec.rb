# typed: false
# frozen_string_literal: true

RSpec.xdescribe "DELETE /v1/@:atname/follow", type: :request, api_version: :v1 do
  # context "アットネームが不正なとき" do
  #   let!(:viewer) { create(:actor, :with_access_token_for_web) }
  #   let!(:oauth_access_token) { viewer.oauth_access_tokens.first }
  #   let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }

  #   it "`422` を返すこと" do
  #     delete("/v1/@unknown/follow", headers:)
  #     expect(response).to have_http_status(:unprocessable_entity)

  #     expected = {
  #       errors: [
  #         {
  #           code: "invalid_input_data",
  #           field: "atname",
  #           message: "atname not found"
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
  #   let!(:target_profile) { create(:user_record).profile_record }
  #   let!(:form) { V1::FollowForm.new(viewer:, target_atname: target_profile.atname) }
  #   let!(:oauth_access_token) { viewer.oauth_access_tokens.first }
  #   let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }

  #   before do
  #     FollowProfileUseCase.new.call(
  #       viewer: form.viewer.not_nil!,
  #       target_profile: form.target_profile.not_nil!
  #     )
  #   end

  #   it "`200` を返すこと" do
  #     expect(FollowRecord.count).to eq(1)

  #     delete("/v1/@#{target_profile.atname}/follow", headers:)
  #     expect(response).to have_http_status(:ok)

  #     expect(FollowRecord.count).to eq(0)

  #     expected = {
  #       profile: build_profile_resource(profile: target_profile, viewer_has_followed: false)
  #     }
  #     actual = JSON.parse(response.body)
  #     expect(actual).to include(expected.deep_stringify_keys)

  #     assert_response_schema_confirm(200)
  #   end
  # end
end
