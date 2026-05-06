# typed: strict
# frozen_string_literal: true

class SuggestedFollowForm < ApplicationForm
  sig { returns(T.nilable(ProfileRecord)) }
  attr_accessor :source_profile

  attribute :target_atname, :string

  validates :source_profile, presence: true
  validates :target_profile, presence: true

  sig { returns(T.nilable(ProfileRecord)) }
  def target_profile
    ProfileRecord.kept.find_by(atname: target_atname)
  end

  # ビューテンプレートとの互換性のためのエイリアス
  sig { returns(T.nilable(String)) }
  def atname
    target_atname
  end
end
