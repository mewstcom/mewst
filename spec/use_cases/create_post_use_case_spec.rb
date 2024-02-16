# typed: false
# frozen_string_literal: true

RSpec.describe CreatePostUseCase do
  let!(:viewer) { create(:actor) }
  let!(:form) { PostForm.new(content: "hello") }
  let!(:use_case) { CreatePostUseCase.new }
  let!(:home_timeline) { instance_spy(Profile::HomeTimeline) }

  before do
    create(:oauth_application, :mewst_web)

    allow(viewer).to receive(:home_timeline).and_return(home_timeline)
    allow(home_timeline).to receive(:add_post)

    allow(FanoutPostJob).to receive(:perform_later)
  end

  it "投稿が作れること" do
    expect(Post.count).to eq(0)

    result = use_case.call(viewer:, content: form.content.not_nil!)

    expect(Post.count).to eq(1)
    post = Post.first
    expect(post.content).to eq("hello")
    expect(viewer.profile.last_post_at).to eq(post.published_at)

    expect(home_timeline).to have_received(:add_post).exactly(1).time
    expect(FanoutPostJob).to have_received(:perform_later).exactly(1).time

    expect(result.post).to eq(post)
  end
end
