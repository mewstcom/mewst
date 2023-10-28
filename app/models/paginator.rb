# typed: strict
# frozen_string_literal: true

class Paginator
  extend T::Sig

  class Result < T::Struct
    const :records, ActiveRecord::Relation
    const :page_info, PageInfo
  end

  sig { params(records: ActiveRecord::Relation).void }
  def initialize(records:)
    @records = records
  end

  sig do
    params(
      before: T.nilable(T::Mewst::DatabaseId),
      after: T.nilable(T::Mewst::DatabaseId),
      limit: Integer
    ).returns(Result)
  end
  def paginate(before: nil, after: nil, limit: 50)
    before_record = before.nil? ? nil : records.find(before)
    after_record = after.nil? ? nil : records.find(after)

    record_ids = if before_record.nil? && after_record.nil?
      records.order(id: :desc).limit(limit + 1).pluck(:id)
    elsif before_record
      records.where(records.arel_table[:id].gt(before_record.id)).order(id: :asc).limit(limit + 1).pluck(:id)
    elsif after_record
      records.where(records.arel_table[:id].lt(after_record.id)).order(id: :desc).limit(limit + 1).pluck(:id)
    end

    page_info = if before_record.nil? && after_record.nil?
      has_next_page = has_page?(record_ids:, limit:)
      end_cursor = cursor(has_next_page:, record_ids:)
      has_previous_page = false
      start_cursor = nil

      PageInfo.new(end_cursor:, has_next_page:, has_previous_page:, start_cursor:)
    elsif before_record
      has_next_page = true
      end_cursor = record_ids.first
      has_previous_page = has_page?(record_ids:, limit:)
      start_cursor = cursor(has_next_page:, record_ids:)

      PageInfo.new(end_cursor:, has_next_page:, has_previous_page:, start_cursor:)
    elsif after_record
      has_next_page = has_page?(record_ids:, limit:)
      end_cursor = cursor(has_next_page:, record_ids:)
      has_previous_page = true
      start_cursor = record_ids.first

      PageInfo.new(end_cursor:, has_next_page:, has_previous_page:, start_cursor:)
    end

    Result.new(
      records: records.where(id: record_ids.first(limit)).order(id: :desc),
      page_info: page_info.not_nil!
    )
  end

  sig { returns(ActiveRecord::Relation) }
  attr_reader :records
  private :records

  sig { params(record_ids: T::Array[T::Mewst::DatabaseId], limit: Integer).returns(T::Boolean) }
  private def has_page?(record_ids:, limit:)
    record_ids.length == (limit + 1)
  end

  sig { params(has_next_page: T::Boolean, record_ids: T::Array[T::Mewst::DatabaseId]).returns(T.nilable(String)) }
  private def cursor(has_next_page:, record_ids:)
    has_next_page ? record_ids[-2] : record_ids.last
  end
end
