# typed: strict
# frozen_string_literal: true

class Services::CreateRepost < Services::Base
  class Result < T::Struct
    const :post, Post
  end

  sig { params(form: Forms::Repost).void }
  def initialize(form:)
    @form = form
  end

  sig { returns(Result) }
  def call
    new_post = ActiveRecord::Base.transaction do
      repost = Repost.create!(repostable: form.post!.postable)
      post = form.profile!.posts.create!(postable: repost, published_at: Time.current)

      post
    end

    form.profile!.home_timeline.add_post(post: new_post)
    FanoutPostJob.perform_async(post_id: new_post.id)

    Result.new(post: new_post)
  end

  private

  sig { returns(Forms::Repost) }
  attr_reader :form
end
