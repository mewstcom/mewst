# typed: false
# frozen_string_literal: true

RSpec.xdescribe "GET /v1/notifications", type: :request, api_version: :v1 do
  context "正常系" do
    let!(:viewer) { create(:actor, :with_access_token_for_web) }
    let!(:oauth_access_token) { viewer.oauth_access_tokens.first }
    let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }
    let!(:other_actor) { create(:actor) }
    let!(:post) { CreatePostUseCase.new.call(viewer:, content: "Hello").post }

    before do
      FollowProfileUseCase.new.call(viewer: other_actor, target_profile: viewer.profile_record)
      CreateStampUseCase.new.call(viewer: other_actor, target_post: post)
    end

    it "通知一覧が返ること" do
      get("/v1/notifications", headers:)

      expect(response).to have_http_status(:ok)
      assert_response_schema_confirm(200)

      expect(viewer.notification_records.count).to eq(1)
      notification = viewer.notification_records.first

      expected = {
        notifications: [
          {
            id: notification.id,
            source_profile: build_profile_resource(profile: other_actor.profile_record, viewer_has_followed: false),
            kind: "Stamp",
            item: {
              source_profile: build_profile_resource(profile: other_actor.profile_record, viewer_has_followed: false),
              target_post: build_post_resource(post: post.reload, viewer_has_followed: false, viewer_has_stamped: false)
            },
            notified_at: notification.notified_at.iso8601
          }
        ],
        page_info: {
          has_next_page: false,
          has_previous_page: false,
          start_cursor: nil,
          end_cursor: notification.id
        }
      }
      actual = JSON.parse(response.body)
      expect(actual).to include(expected.deep_stringify_keys)
    end
  end
end
