# typed: strict
# frozen_string_literal: true

module TimeHelper
  extend T::Sig

  sig { params(time: ActiveSupport::TimeWithZone).returns(String) }
  def mst_absolute_time(time)
    time.in_time_zone(current_actor&.time_zone).to_fs(:ymdhm)
  end

  sig { params(from_time: ActiveSupport::TimeWithZone).returns(String) }
  def mst_time_ago_in_words(from_time)
    days = (Time.current.to_date - from_time.to_date).to_i

    if days > 3
      return mst_absolute_time(from_time)
    end

    time_ago_in_words(from_time)
  end

  sig { params(time_zone: ActiveSupport::TimeZone).returns(String) }
  def time_zone_name(time_zone)
    offset = time_zone.utc_offset / 3600
    formatted_offset = format("%02d:00", offset.abs)
    formatted_offset = (offset >= 0) ? "+#{formatted_offset}" : "-#{formatted_offset}"
    "(GMT#{formatted_offset}) #{time_zone.name}"
  end
end
