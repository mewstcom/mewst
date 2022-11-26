# typed: strict
# frozen_string_literal: true

class User < ApplicationRecord
  include Profilable

  belongs_to :account

  delegate :follows, :idname, to: :profile

  sig { params(target_profile: Profile).returns(T::Boolean) }
  def my_profile?(target_profile:)
    idname == target_profile.idname
  end

  sig { params(target_profile: Profile).returns(T::Boolean) }
  def following?(target_profile:)
    follows.exists?(target_profile:)
  end
end
