# typed: strict
# frozen_string_literal: true

class ProfileRecord::Postability
  extend T::Sig

  sig { params(profile: ProfileRecord).void }
  def initialize(profile:)
    @profile = profile
  end

  sig { params(post: PostRecord).void }
  def delete_post(post:)
    ActiveRecord::Base.transaction do
      post.destroy!
    end
  end

  sig { returns(ProfileRecord) }
  attr_reader :profile
  private :profile
end
