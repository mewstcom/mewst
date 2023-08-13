# typed: false
# frozen_string_literal: true

RSpec.describe CreateCommentPostService do
  let!(:profile) { create(:profile, :for_user) }
  let!(:form) { Latest::CommentPostForm.new(profile:, comment: "hello") }
  let!(:input) { CreateCommentPostService::Input.from_latest_form(form:) }
  let!(:service) { CreateCommentPostService.new }
  let!(:home_timeline) { instance_spy(Profile::HomeTimeline) }

  before do
    allow(profile).to receive(:home_timeline).and_return(home_timeline)
    allow(home_timeline).to receive(:add_post)

    allow(FanoutPostJob).to receive(:perform_async)
  end

  it "creates a new comment post" do
    expect(CommentPost.count).to eq(0)
    expect(Post.count).to eq(0)

    result = service.call(input:)

    expect(CommentPost.count).to eq(1)
    comment_post = CommentPost.first
    expect(comment_post.comment).to eq("hello")

    expect(Post.count).to eq(1)
    post = Post.first
    expect(post.comment_post!).to eq(comment_post)

    expect(home_timeline).to have_received(:add_post).exactly(1).time
    expect(FanoutPostJob).to have_received(:perform_async).exactly(1).time

    expect(result.post).to eq(post)
  end
end
