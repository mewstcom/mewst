# typed: strict
# frozen_string_literal: true

class Commands::CreateCommentedPost < Commands::Base
  Result = Data.define(:post)

  sig { params(form: Forms::CommentedPost).void }
  def initialize(form:)
    @form = form
  end

  sig { returns(Result) }
  def call
    new_post = ActiveRecord::Base.transaction do
      commented_post = CommentedPost.create!(comment: form.comment)
      post = form.profile.posts.create!(postable: commented_post, published_at: Time.current)
      post
    end

    form.profile.home_timeline.add_post(post: new_post)
    FanoutPostJob.perform_async(post_id: new_post.id)

    Result.new(post: new_post)
  end

  private

  sig { returns(Forms::CommentedPost) }
  attr_reader :form
end
