# typed: strict
# frozen_string_literal: true

class User < ApplicationRecord
  delegate :follows, :idname, to: :profile

  has_many :email_confirmations, dependent: :restrict_with_exception
  has_one :user_profile, dependent: :restrict_with_exception
  has_one :profile, dependent: :restrict_with_exception, through: :user_profile

  validates :email, email: true, presence: true, uniqueness: true

  sig { returns(T::Boolean) }
  def track_sign_in
    old_current, new_current = current_signed_in_at, Time.current

    self.last_signed_in_at = old_current || new_current
    self.current_signed_in_at = new_current
    self.sign_in_count += 1

    save!(validate: false)
  end

  sig { params(target_profile: Profile).returns(T::Boolean) }
  def my_profile?(target_profile:)
    idname == target_profile.idname
  end

  sig { params(target_profile: Profile).returns(T::Boolean) }
  def following?(target_profile:)
    follows.exists?(target_profile:)
  end
end
