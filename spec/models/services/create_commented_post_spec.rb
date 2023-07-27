# typed: false
# frozen_string_literal: true

RSpec.describe Services::CreateCommentedPost do
  let!(:profile) { create(:profile, :with_member) }
  let!(:form) { Forms::CommentedPost.new(profile:, comment: "hello") }
  let!(:command) { Services::CreateCommentedPost.new(form:) }
  let!(:home_timeline) { instance_spy(Profile::HomeTimeline) }

  before do
    allow(profile).to receive(:home_timeline).and_return(home_timeline)
    allow(home_timeline).to receive(:add_post)

    allow(FanoutPostJob).to receive(:perform_async)
  end

  it "creates a new commented post" do
    expect(CommentedPost.count).to eq(0)
    expect(Post.count).to eq(0)

    result = command.call

    expect(CommentedPost.count).to eq(1)
    commented_post = CommentedPost.first
    expect(commented_post.comment).to eq("hello")

    expect(Post.count).to eq(1)
    post = Post.first
    expect(post.postable).to eq(commented_post)

    expect(home_timeline).to have_received(:add_post).exactly(1).time
    expect(FanoutPostJob).to have_received(:perform_async).exactly(1).time

    expect(result.post).to eq(post)
  end
end
