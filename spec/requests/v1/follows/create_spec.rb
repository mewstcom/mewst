# typed: false
# frozen_string_literal: true

RSpec.xdescribe "POST /v1/@:atname/follow", type: :request, api_version: :v1 do
  context "アットネームが正しくないとき" do
    let!(:viewer) { create(:actor, :with_access_token_for_web) }
    let!(:profile) { viewer.profile_record }
    let!(:oauth_access_token) { viewer.oauth_access_tokens.first }
    let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }

    it "`422` を返すこと" do
      post("/v1/@#{profile.atname}/follow", headers:)
      expect(response).to have_http_status(:unprocessable_entity)

      expected = {
        errors: [
          {
            code: "invalid_input_data",
            field: "atname",
            message: "atname is myself. Cannot follow myself"
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
    let!(:target_profile) { create(:user_record).profile_record }
    let!(:oauth_access_token) { viewer.oauth_access_tokens.first }
    let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }

    it "`201` を返すこと" do
      expect(FollowRecord.count).to eq(0)

      post("/v1/@#{target_profile.atname}/follow", headers:)
      expect(response).to have_http_status(:created)

      expect(FollowRecord.count).to eq(1)

      expected = {
        profile: build_profile_resource(profile: target_profile, viewer_has_followed: true)
      }
      actual = JSON.parse(response.body)
      expect(actual).to include(expected.deep_stringify_keys)

      assert_response_schema_confirm(201)
    end
  end
end
