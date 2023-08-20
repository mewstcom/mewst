# typed: false
# frozen_string_literal: true

RSpec.describe CreatePostService do
  let!(:user) { create(:user) }
  let!(:profile) { user.profile }
  let!(:form) { Latest::PostForm.new(profile:, comment: "hello") }
  let!(:input) { CreatePostService::Input.from_latest_form(form:) }
  let!(:service) { CreatePostService.new }
  let!(:home_timeline) { instance_spy(Profile::HomeTimeline) }

  before do
    allow(profile).to receive(:home_timeline).and_return(home_timeline)
    allow(home_timeline).to receive(:add_post)

    allow(FanoutPostJob).to receive(:perform_async)
  end

  it "creates a new comment post" do
    expect(Post.count).to eq(0)

    result = service.call(input:)

    expect(Post.count).to eq(1)
    post = Post.first
    expect(post.comment).to eq("hello")

    expect(home_timeline).to have_received(:add_post).exactly(1).time
    expect(FanoutPostJob).to have_received(:perform_async).exactly(1).time

    expect(result.post).to eq(post)
  end
end
