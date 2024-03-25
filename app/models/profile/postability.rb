# typed: strict
# frozen_string_literal: true

class Profile::Postability
  extend T::Sig

  sig { params(profile: Profile).void }
  def initialize(profile:)
    @profile = profile
  end

  sig { params(post: Post).void }
  def delete_post(post:)
    ActiveRecord::Base.transaction do
      post.destroy!
    end
  end

  sig { returns(Profile) }
  attr_reader :profile
  private :profile
end
