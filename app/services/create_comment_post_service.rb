# typed: strict
# frozen_string_literal: true

class CreateCommentPostService < ApplicationService
  class Result < T::Struct
    const :post, Post
  end

  sig { params(form: Forms::CommentPost).void }
  def initialize(form:)
    @form = form
  end

  sig { returns(Result) }
  def call
    post = form.profile!.posts.create!(kind: :comment_post, published_at: Time.current)
    post.create_comment_post!(comment: form.comment)

    form.profile!.home_timeline.add_post(post:)
    FanoutPostJob.perform_async(post_id: post.id)

    Result.new(post:)
  end

  sig { returns(Forms::CommentPost) }
  attr_reader :form
  private :form
end
