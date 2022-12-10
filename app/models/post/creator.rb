# typed: strict
# frozen_string_literal: true

class Post::Creator
  extend T::Sig

  include ActiveModel::Model

  validates :content, length: {maximum: Post::CONTENT_MAXIMUM_LENGTH}, presence: true

  sig { params(profile: Profile, content: T.nilable(String)).void }
  def initialize(profile:, content:)
    @profile = profile
    @content = content
  end

  sig { returns(T.self_type) }
  def call
    post = profile.posts.create!(content:)

    FanOutPostJob.perform_async(post.id)

    self
  end

  private

  sig { returns(Profile) }
  attr_reader :profile

  sig { returns(T.nilable(String)) }
  attr_reader :content
end
