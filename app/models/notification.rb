# typed: strict
# frozen_string_literal: true

class Notification < ApplicationRecord
  extend Enumerize

  delegated_type :notifiable, types: (
    NotifiableType.values.map(&:serialize) +
    # HACK: 2つ以上のモデル名を指定しないと型エラーになるため、notifiable では必要ないがNilClassを追加している
    # https://wikino.app/s/shimbaco/pages/345
    ["NilClass"]
  )

  belongs_to :source_profile, class_name: "Profile"
  belongs_to :target_profile, class_name: "Profile"

  sig { returns(NotifiableType) }
  def deserialized_notifiable_type
    NotifiableType.deserialize(notifiable_type)
  end
end
