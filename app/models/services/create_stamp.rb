# typed: strict
# frozen_string_literal: true

class Services::CreateStamp < Services::Base
  class Result < T::Struct
    const :post, Post
  end

  sig { params(form: Forms::Stamp).void }
  def initialize(form:)
    @form = form
  end

  sig { returns(Result) }
  def call
    comment_post = form.original_post.comment_post!
    form.profile!.stamps.where(comment_post:).first_or_create!(stamped_at: Time.current)

    Result.new(post: form.target_post!)
  end

  sig { returns(Forms::Stamp) }
  attr_reader :form
  private :form
end
