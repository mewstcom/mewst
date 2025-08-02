# typed: false
# frozen_string_literal: true

RSpec.xdescribe "POST /v1/suggested_profiles/@:atname/check", type: :request, api_version: :v1 do
  # it "アットネームが正しくないとき、422を返すこと" do
  #   viewer = FactoryBot.create(:actor, :with_access_token_for_web)
  #   oauth_access_token = viewer.oauth_access_tokens.first
  #   headers = {"Authorization" => "bearer #{oauth_access_token.token}"}

  #   post("/v1/suggested_profiles/@unknown/check", headers:)
  #   expect(response).to have_http_status(:unprocessable_entity)

  #   expected = {
  #     errors: [
  #       {
  #         code: "invalid_input_data",
  #         field: "atname",
  #         message: "atname not found"
  #       }
  #     ]
  #   }
  #   actual = JSON.parse(response.body)
  #   expect(actual).to include(expected.deep_stringify_keys)

  #   assert_response_schema_confirm(422)
  # end

  # it "入力データが正しいとき、204を返すこと" do
  #   viewer = FactoryBot.create(:actor, :with_access_token_for_web)
  #   oauth_access_token = viewer.oauth_access_tokens.first
  #   headers = {"Authorization" => "bearer #{oauth_access_token.token}"}
  #   actor_a = FactoryBot.create(:actor)
  #   actor_b = FactoryBot.create(:actor)

  #   FollowProfileUseCase.new.call(source_profile: viewer.profile_record, target_profile: actor_a.profile_record)
  #   FollowProfileUseCase.new.call(source_profile: actor_a.profile_record, target_profile: actor_b.profile_record)
  #   viewer.profile_record.create_suggested_follows!

  #   expect(viewer.profile_record.suggested_follow_records.count).to eq(1)
  #   suggested_follow = viewer.suggested_follow_records.first
  #   expect(suggested_follow.target_profile).to eq(actor_b.profile_record)
  #   expect(suggested_follow.checked_at).to be_nil

  #   post("/v1/suggested_profiles/@#{actor_b.atname}/check", headers:)
  #   expect(response).to have_http_status(:no_content)

  #   expect(viewer.suggested_follow_records.count).to eq(1)
  #   suggested_follow = viewer.profile_record.suggested_follow_records.first
  #   expect(suggested_follow.target_profile).to eq(actor_b.profile_record)
  #   expect(suggested_follow.checked_at).not_to be_nil

  #   assert_response_schema_confirm(204)
  # end
end
