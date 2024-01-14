# typed: false
# frozen_string_literal: true

RSpec.describe "POST /v1/suggested_profiles/@:atname/check", type: :request, api_version: :v1 do
  context "アットネームが正しくないとき" do
    let!(:viewer) { create(:actor, :with_access_token_for_web) }
    let!(:oauth_access_token) { viewer.oauth_access_tokens.first }
    let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }

    it "`422` を返すこと" do
      post("/v1/suggested_profiles/@unknown/check", headers:)
      expect(response).to have_http_status(:unprocessable_entity)

      expected = {
        errors: [
          {
            code: "invalid_input_data",
            field: "atname",
            message: "atname not found"
          }
        ]
      }
      actual = JSON.parse(response.body)
      expect(actual).to include(expected.deep_stringify_keys)

      assert_response_schema_confirm(422)
    end
  end

  context "入力データが正しいとき" do
    let!(:viewer) { create(:actor, :with_access_token_for_web) }
    let!(:oauth_access_token) { viewer.oauth_access_tokens.first }
    let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }
    let!(:actor_a) { create(:actor) }
    let!(:actor_b) { create(:actor) }

    before do
      FollowProfileUseCase.new.call(viewer:, target_profile: actor_a.profile)
      FollowProfileUseCase.new.call(viewer: actor_a, target_profile: actor_b.profile)
      viewer.profile.create_suggested_follows!
    end

    it "`204` を返すこと" do
      expect(viewer.profile.suggested_follows.count).to eq(1)
      suggested_follow = viewer.suggested_follows.first
      expect(suggested_follow.target_profile).to eq(actor_b.profile)
      expect(suggested_follow.checked_at).to be_nil

      post("/v1/suggested_profiles/@#{actor_b.atname}/check", headers:)
      expect(response).to have_http_status(:no_content)

      expect(viewer.suggested_follows.count).to eq(1)
      suggested_follow = viewer.profile.suggested_follows.first
      expect(suggested_follow.target_profile).to eq(actor_b.profile)
      expect(suggested_follow.checked_at).not_to be_nil

      assert_response_schema_confirm(204)
    end
  end
end
