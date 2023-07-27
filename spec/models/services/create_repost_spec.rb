# typed: false
# frozen_string_literal: true

RSpec.describe Services::CreateRepost do
  let!(:profile) { create(:profile, :with_member) }
  let!(:commented_post_form) { Forms::CommentedPost.new(profile:, comment: "hello") }
  let!(:target_post) { Services::CreateCommentedPost.new(form: commented_post_form).call.post }
  let!(:form) { Forms::Repost.new(profile:, post_id: target_post.id) }
  let!(:command) { Services::CreateRepost.new(form:) }
  let!(:home_timeline) { instance_spy(Profile::HomeTimeline) }

  before do
    allow(profile).to receive(:home_timeline).and_return(home_timeline)
    allow(home_timeline).to receive(:add_post)

    allow(FanoutPostJob).to receive(:perform_async)
  end

  it "creates a new commented post" do
    expect(Repost.count).to eq(0)
    expect(Post.count).to eq(1)

    result = command.call

    expect(Repost.count).to eq(1)
    repost = Repost.first
    expect(repost.repostable).to eq(target_post.postable)

    expect(Post.count).to eq(2)
    post = Post.where.not(id: target_post.id).first
    expect(post.postable).to eq(repost)

    expect(home_timeline).to have_received(:add_post).exactly(1).time
    expect(FanoutPostJob).to have_received(:perform_async).exactly(1).time

    expect(result.post).to eq(post)
  end
end
