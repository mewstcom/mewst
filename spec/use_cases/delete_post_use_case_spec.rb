# typed: false
# frozen_string_literal: true

RSpec.describe DeletePostUseCase do
  context "正常系" do
    def setup_data
      FactoryBot.create(:oauth_application, :mewst_web)
      link = FactoryBot.create(:link)
      post_author = FactoryBot.create(:actor)
      result = CreatePostUseCase.new.call(viewer: post_author, content: "hello", canonical_url: link.canonical_url)
      target_post = result.post
      current_actor = FactoryBot.create(:actor)
      CreateStampUseCase.new.call(current_actor:, target_post:)

      {target_post:}
    end

    it "ポストが削除できること" do
      setup_data => {target_post:}

      expect(Link.count).to eq(1)
      expect(Post.count).to eq(1)
      expect(Stamp.count).to eq(1)
      expect(Notification.count).to eq(1)
      expect(PostLink.count).to eq(1)
      expect(HomeTimelinePost.count).to eq(1)

      DeletePostUseCase.new.call(target_post:)

      expect(Link.count).to eq(1)
      expect(Post.count).to eq(0)
      expect(Stamp.count).to eq(0)
      expect(Notification.count).to eq(0)
      expect(PostLink.count).to eq(0)
      expect(HomeTimelinePost.count).to eq(0)
    end
  end
end
