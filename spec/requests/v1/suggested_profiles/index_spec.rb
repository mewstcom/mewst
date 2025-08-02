# typed: false
# frozen_string_literal: true

RSpec.xdescribe "GET /v1/suggested_profiles", type: :request, api_version: :v1 do
  # context "正常系" do
  #   let!(:viewer) { create(:actor, :with_access_token_for_web) }
  #   let!(:oauth_access_token) { viewer.oauth_access_tokens.first }
  #   let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }
  #   let!(:actor_a) { create(:actor) }
  #   let!(:actor_b) { create(:actor) }

  #   before do
  #     FollowProfileUseCase.new.call(viewer:, target_profile: actor_a.profile_record)
  #     FollowProfileUseCase.new.call(viewer: actor_a, target_profile: actor_b.profile_record)
  #     viewer.profile_record.create_suggested_follows!
  #   end

  #   it "おすすめプロフィール一覧が返ること" do
  #     get("/v1/suggested_profiles", headers:)

  #     expect(response).to have_http_status(:ok)
  #     assert_response_schema_confirm(200)

  #     expect(viewer.suggested_followees.count).to eq(1)
  #     suggested_profile = viewer.suggested_followees.first

  #     expected = {
  #       profiles: [
  #         build_profile_resource(profile: suggested_profile, viewer_has_followed: false)
  #       ]
  #     }
  #     actual = JSON.parse(response.body)
  #     expect(actual).to include(expected.deep_stringify_keys)
  #   end
  # end
end
