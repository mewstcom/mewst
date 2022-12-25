# typed: strict
# frozen_string_literal: true

module TimeHelper
  extend T::Sig

  sig { params(from_time: ActiveSupport::TimeWithZone).returns(String) }
  def mst_time_ago_in_words(from_time)
    days = (Time.current.to_date - from_time.to_date).to_i

    return from_time.to_fs(:ymdhm) if days > 3

    spacer = (I18n.locale == :en) ? " " : ""
    "#{time_ago_in_words(from_time)}#{spacer}#{I18n.t("nouns.ago").downcase}"
  end
end
