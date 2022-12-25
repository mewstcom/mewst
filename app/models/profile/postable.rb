# typed: strict
# frozen_string_literal: true

module Profile::Postable
  extend T::Sig
  extend ActiveSupport::Concern

  included do
    has_many :posts, dependent: :restrict_with_exception
  end

  sig { params(content: T.nilable(String)).returns(Post) }
  def create_post(content:)
    post = posts.create!(content:)

    FanOutPostJob.perform_async(post.id)

    post
  end
end
