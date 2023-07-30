# typed: strict
# frozen_string_literal: true

class Services::CreateCommentedPost < Services::Base
  class Result < T::Struct
    const :post, Post
  end

  sig { params(form: Forms::CommentedPost).void }
  def initialize(form:)
    @form = form
  end

  sig { returns(Result) }
  def call
    commented_post = CommentedPost.create!(comment: form.comment)
    post = form.profile!.posts.create!(postable: commented_post, published_at: Time.current)

    form.profile!.home_timeline.add_post(post:)
    FanoutPostJob.perform_async(post_id: post.id)

    Result.new(post:)
  end

  private

  sig { returns(Forms::CommentedPost) }
  attr_reader :form
end
