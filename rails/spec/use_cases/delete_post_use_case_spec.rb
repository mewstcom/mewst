# typed: false
# frozen_string_literal: true

RSpec.describe DeletePostUseCase do
  context "正常系" do
    def setup_data
      link = FactoryBot.create(:link_record)
      post_author = FactoryBot.create(:actor)
      target_post = FactoryBot.create(:post_record, profile_record: post_author.profile_record)
      target_post.create_post_link_record!(link_record: link)
      post_author.profile_record.home_timeline.add_post!(post: target_post)
      viewer = FactoryBot.create(:actor)
      CreateStampUseCase.new.call(viewer:, target_post:)

      {target_post:}
    end

    it "ポストが削除できること" do
      setup_data => {target_post:}

      expect(LinkRecord.count).to eq(1)
      expect(PostRecord.count).to eq(1)
      expect(StampRecord.count).to eq(1)
      expect(NotificationRecord.count).to eq(1)
      expect(PostLinkRecord.count).to eq(1)
      expect(HomeTimelinePostRecord.count).to eq(1)

      DeletePostUseCase.new.call(target_post:)

      expect(LinkRecord.count).to eq(1)
      expect(PostRecord.count).to eq(0)
      expect(StampRecord.count).to eq(0)
      expect(NotificationRecord.count).to eq(0)
      expect(PostLinkRecord.count).to eq(0)
      expect(HomeTimelinePostRecord.count).to eq(0)
    end
  end
end
