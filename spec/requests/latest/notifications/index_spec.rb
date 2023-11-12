# typed: false
# frozen_string_literal: true

RSpec.describe "GET /latest/notifications", type: :request, api_version: :latest do
  context "正常系" do
    let!(:viewer) { create(:actor, :with_access_token_for_web) }
    let!(:oauth_access_token) { viewer.oauth_access_tokens.first }
    let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }
    let!(:other_actor) { create(:actor) }
    let!(:post) { CreatePostUseCase.new.call(viewer:, content: "Hello").post }

    before do
      FollowProfileUseCase.new.call(viewer: other_actor, target_profile: viewer.profile)
      CreateStampUseCase.new.call(viewer: other_actor, target_post: post)
    end

    it "通知一覧が返ること" do
      get("/latest/notifications", headers:)

      expect(response).to have_http_status(:ok)
      assert_response_schema_confirm(200)

      expect(viewer.notifications.count).to eq(2)
      notification_1 = viewer.notifications.last
      notification_2 = viewer.notifications.first

      expected = {
        notifications: [
          {
            id: notification_1.id,
            source_profile: build_profile_resource(profile: other_actor.profile, viewer_has_followed: false),
            kind: "Stamp",
            item: {
              source_profile: build_profile_resource(profile: other_actor.profile, viewer_has_followed: false),
              target_post: build_post_resource(post: post.reload, viewer_has_followed: false, viewer_has_stamped: false)
            },
            notified_at: notification_1.notified_at.iso8601
          },
          {
            id: notification_2.id,
            source_profile: build_profile_resource(profile: other_actor.profile, viewer_has_followed: false),
            kind: "Follow",
            item: {
              source_profile: build_profile_resource(profile: other_actor.profile, viewer_has_followed: false)
            },
            notified_at: notification_2.notified_at.iso8601
          }
        ],
        page_info: {
          has_next_page: false,
          has_previous_page: false,
          start_cursor: nil,
          end_cursor: notification_2.id
        }
      }
      actual = JSON.parse(response.body)
      expect(actual).to include(expected.deep_stringify_keys)
    end
  end
end
