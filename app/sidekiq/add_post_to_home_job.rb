# typed: strict
# frozen_string_literal: true

class AddPostToHomeJob
  extend T::Sig

  include Sidekiq::Job

  sig { params(post_id: String, profile_id: String).returns(T::Boolean) }
  def perform(post_id, profile_id)
    form = PostToHomeForm.new(post_id:, profile_id:)

    if form.valid?
      AddPostToHomeService.new(form:).call
    end

    true
  rescue ActiveRecord::RecordNotFound
    true
  end
end
