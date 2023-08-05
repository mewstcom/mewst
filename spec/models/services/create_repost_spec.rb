# typed: false
# frozen_string_literal: true

RSpec.describe Services::CreateRepost do
  let!(:profile_1) { create(:profile, :for_user) }
  let!(:profile_2) { create(:profile, :for_user) }
  let!(:comment_post_form) { Forms::CommentPost.new(profile: profile_1, comment: "hello") }
  let!(:target_post) { Services::CreateCommentPost.new(form: comment_post_form).call.post }
  let!(:form) { Forms::Repost.new(profile: profile_2, post_id: target_post.id) }
  let!(:service) { Services::CreateRepost.new(form:) }
  let!(:home_timeline) { instance_spy(Profile::HomeTimeline) }

  before do
    profile_2.follow(target_profile: profile_1)

    allow(profile_2).to receive(:home_timeline).and_return(home_timeline)
    allow(home_timeline).to receive(:add_post)

    allow(FanoutPostJob).to receive(:perform_async)
  end

  it "creates a new comment post" do
    expect(Repost.count).to eq(0)
    expect(Post.count).to eq(1)

    result = service.call

    expect(Post.count).to eq(2)
    post = Post.where.not(id: target_post.id).first
    expect(post.kind).to eq(:repost)

    expect(target_post.reload.reposts_count).to eq(1)

    expect(Repost.count).to eq(1)
    repost = Repost.first
    expect(repost.post).to eq(post)
    expect(repost.target_post).to eq(target_post)
    expect(repost.target_profile).to eq(profile_1)
    expect(repost.original_post).to eq(target_post)
    expect(repost.original_profile).to eq(profile_1)

    expect(home_timeline).to have_received(:add_post).exactly(1).time
    expect(FanoutPostJob).to have_received(:perform_async).exactly(1).time

    expect(result.post).to eq(post)
  end
end
