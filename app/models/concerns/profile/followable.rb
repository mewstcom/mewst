# typed: true
# frozen_string_literal: true

module Profile::Followable
  extend T::Sig
  extend ActiveSupport::Concern

  included do
    has_many :follows, dependent: :restrict_with_exception, foreign_key: :source_profile_id, inverse_of: :source_profile
    has_many :inverse_follows, class_name: "Follow", dependent: :restrict_with_exception, foreign_key: :target_profile_id, inverse_of: :target_profile
    has_many :followees, class_name: "Profile", source: :target_profile, through: :follows
    has_many :followers, class_name: "Profile", source: :source_profile, through: :inverse_follows
  end

  sig { params(target_profile: Profile).returns(T.self_type) }
  def follow(target_profile:)
    return self if me?(target_profile:)

    follow = follows.where(target_profile:).first_or_initialize
    follow.save!

    self
  end

  sig { params(target_profile: Profile).returns(T.self_type) }
  def unfollow(target_profile:)
    return self if me?(target_profile:)

    follow = follows.find_by(target_profile:)
    follow&.destroy!

    self
  end
end
